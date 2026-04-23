package output

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/imryanparsa/kfqdn/internal/resolver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/kubernetes"
)

// PrintResults prints DNS results for a single named resource in kubectl-style table format.
func PrintResults(streams genericiooptions.IOStreams, name string, results []resolver.Result, doResolve bool) {
	w := printers.GetNewTabWriter(streams.Out)
	defer w.Flush()

	if doResolve {
		fmt.Fprintln(w, "NAME\tDNS NAME\tKIND\tIP(S)")
	} else {
		fmt.Fprintln(w, "NAME\tDNS NAME\tKIND")
	}

	for _, r := range results {
		if doResolve {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", name, r.Name, r.Kind, lookupHost(r.Name))
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\n", name, r.Name, r.Kind)
		}
	}
}

// ListAll prints DNS names for every Service across all namespaces.
func ListAll(ctx context.Context, client kubernetes.Interface, streams genericiooptions.IOStreams, domain string, doResolve bool) error {
	svcs, err := client.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing services: %w", err)
	}

	w := printers.GetNewTabWriter(streams.Out)
	defer w.Flush()

	if doResolve {
		fmt.Fprintln(w, "NAMESPACE\tNAME\tDNS NAME\tKIND\tIP(S)")
	} else {
		fmt.Fprintln(w, "NAMESPACE\tNAME\tDNS NAME\tKIND")
	}

	for i := range svcs.Items {
		svc := &svcs.Items[i]
		for _, r := range resolver.ServiceResultsFor(svc, domain) {
			if doResolve {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", svc.Namespace, svc.Name, r.Name, r.Kind, lookupHost(r.Name))
			} else {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", svc.Namespace, svc.Name, r.Name, r.Kind)
			}
		}
	}
	return nil
}

// PrintNamedResults prints a list of resources in kubectl-style table format.
// withNamespace shows the NAMESPACE column; the TYPE column appears automatically
// when NamedResults entries have a non-empty Type field.
func PrintNamedResults(streams genericiooptions.IOStreams, named []resolver.NamedResults, withNamespace, doResolve bool) {
	w := printers.GetNewTabWriter(streams.Out)
	defer w.Flush()

	withType := len(named) > 0 && named[0].Type != ""

	// Build header.
	header := ""
	if withNamespace {
		header += "NAMESPACE\t"
	}
	if withType {
		header += "TYPE\t"
	}
	header += "NAME\tDNS NAME\tKIND"
	if doResolve {
		header += "\tIP(S)"
	}
	fmt.Fprintln(w, header)

	for _, nr := range named {
		for _, r := range nr.Results {
			prefix := ""
			if withNamespace {
				prefix += nr.Namespace + "\t"
			}
			if withType {
				prefix += nr.Type + "\t"
			}
			if doResolve {
				fmt.Fprintf(w, "%s%s\t%s\t%s\t%s\n", prefix, nr.Name, r.Name, r.Kind, lookupHost(r.Name))
			} else {
				fmt.Fprintf(w, "%s%s\t%s\t%s\n", prefix, nr.Name, r.Name, r.Kind)
			}
		}
	}
}

func lookupHost(name string) string {
	addrs, err := net.LookupHost(name)
	if err != nil {
		return "[unresolved]"
	}
	return strings.Join(addrs, ", ")
}
