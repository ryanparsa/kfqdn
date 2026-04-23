package output

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/imryanparsa/kfqdn/internal/resolver"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/cli-runtime/pkg/printers"
)

// jsonRow is the structure emitted for each result row when --output=json.
type jsonRow struct {
	Namespace string `json:"namespace,omitempty"`
	Type      string `json:"type,omitempty"`
	Name      string `json:"name"`
	DNSName   string `json:"dnsName"`
	Kind      string `json:"kind"`
	IPs       string `json:"ips,omitempty"`
	Extra     string `json:"extra,omitempty"`
}

// PrintResults prints DNS results for a single named resource in kubectl-style table format.
func PrintResults(streams genericiooptions.IOStreams, name string, results []resolver.Result, doResolve bool, outputFormat string) {
	// Wrap in a NamedResults slice to share the common printing path.
	named := []resolver.NamedResults{{Name: name, Results: results}}
	PrintNamedResults(streams, named, false, doResolve, outputFormat)
}

// PrintNamedResults prints a list of resources in the requested output format.
// withNamespace shows the NAMESPACE column; the TYPE column appears automatically
// when any NamedResults entry has a non-empty Type field.
func PrintNamedResults(streams genericiooptions.IOStreams, named []resolver.NamedResults, withNamespace, doResolve bool, outputFormat string) {
	// Determine whether the TYPE column is needed by scanning all entries.
	withType := false
	for _, nr := range named {
		if nr.Type != "" {
			withType = true
			break
		}
	}

	// Determine whether the EXTRA (wide) column is needed.
	withExtra := outputFormat == "wide"

	// Pre-resolve all DNS names concurrently when --resolve is requested.
	var ipMap map[string]string
	if doResolve {
		ipMap = resolveAll(named)
	}

	switch outputFormat {
	case "json":
		printJSON(streams, named, withNamespace, withType, withExtra, doResolve, ipMap)
	default:
		printTable(streams, named, withNamespace, withType, withExtra, doResolve, ipMap)
	}
}

func printTable(streams genericiooptions.IOStreams, named []resolver.NamedResults, withNamespace, withType, withExtra, doResolve bool, ipMap map[string]string) {
	w := printers.GetNewTabWriter(streams.Out)
	defer w.Flush()

	// Build header.
	var cols []string
	if withNamespace {
		cols = append(cols, "NAMESPACE")
	}
	if withType {
		cols = append(cols, "TYPE")
	}
	cols = append(cols, "NAME", "DNS NAME", "KIND")
	if withExtra {
		cols = append(cols, "EXTRA")
	}
	if doResolve {
		cols = append(cols, "IP(S)")
	}
	fmt.Fprintln(w, strings.Join(cols, "\t"))

	for _, nr := range named {
		for _, r := range nr.Results {
			var row []string
			if withNamespace {
				row = append(row, nr.Namespace)
			}
			if withType {
				row = append(row, nr.Type)
			}
			row = append(row, nr.Name, r.Name, r.Kind)
			if withExtra {
				row = append(row, nr.Extra)
			}
			if doResolve {
				row = append(row, ipMap[r.Name])
			}
			fmt.Fprintln(w, strings.Join(row, "\t"))
		}
	}
}

func printJSON(streams genericiooptions.IOStreams, named []resolver.NamedResults, withNamespace, withType, withExtra, doResolve bool, ipMap map[string]string) {
	var rows []jsonRow
	for _, nr := range named {
		for _, r := range nr.Results {
			row := jsonRow{
				Name:    nr.Name,
				DNSName: r.Name,
				Kind:    r.Kind,
			}
			if withNamespace {
				row.Namespace = nr.Namespace
			}
			if withType {
				row.Type = nr.Type
			}
			if withExtra {
				row.Extra = nr.Extra
			}
			if doResolve {
				row.IPs = ipMap[r.Name]
			}
			rows = append(rows, row)
		}
	}
	if rows == nil {
		rows = []jsonRow{}
	}
	enc := json.NewEncoder(streams.Out)
	enc.SetIndent("", "  ")
	_ = enc.Encode(rows)
}

// resolveAll performs parallel DNS lookups for all unique names found in named.
// It returns a map from DNS name to resolved IPs string.
func resolveAll(named []resolver.NamedResults) map[string]string {
	// Collect unique names.
	seen := make(map[string]struct{})
	for _, nr := range named {
		for _, r := range nr.Results {
			seen[r.Name] = struct{}{}
		}
	}

	results := make(map[string]string, len(seen))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for name := range seen {
		name := name
		wg.Add(1)
		go func() {
			defer wg.Done()
			ips := lookupHost(name)
			mu.Lock()
			results[name] = ips
			mu.Unlock()
		}()
	}
	wg.Wait()
	return results
}

func lookupHost(name string) string {
	addrs, err := net.LookupHost(name)
	if err != nil {
		return "[unresolved]"
	}
	return strings.Join(addrs, ", ")
}
