package cli

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/imryanparsa/kfqdn/internal/output"
	"github.com/imryanparsa/kfqdn/internal/resolver"
	"golang.org/x/sync/errgroup"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/client-go/kubernetes"
)

func run(configFlags *genericclioptions.ConfigFlags, streams genericiooptions.IOStreams, args []string, allNamespaces, doResolve bool, timeout, outputFormat, selector string) error {
	restConfig, err := configFlags.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("loading kubeconfig: %w", err)
	}

	ns, _, err := configFlags.ToRawKubeConfigLoader().Namespace()
	if err != nil || ns == "" {
		ns = "default"
	}

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	dur, err := time.ParseDuration(timeout)
	if err != nil {
		return fmt.Errorf("invalid --timeout value %q: %w", timeout, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), dur)
	defer cancel()

	domain := resolver.ClusterDomain(ctx, client)

	// Validate output format early.
	switch outputFormat {
	case "table", "wide", "json":
	default:
		return fmt.Errorf("unknown output format %q: supported formats are table, wide, json", outputFormat)
	}

	listNs := ns
	if allNamespaces {
		listNs = ""
	}

	// No type given (only -A): treat as "all" across all namespaces.
	if len(args) == 0 {
		return listAllTypes(ctx, client, streams, listNs, domain, selector, allNamespaces, doResolve, outputFormat)
	}

	resType := strings.ToLower(args[0])

	// "all" aggregates every resource type.
	if resType == "all" && len(args) == 1 {
		return listAllTypes(ctx, client, streams, listNs, domain, selector, allNamespaces, doResolve, outputFormat)
	}

	res, ok := resolver.Registry[resType]
	if !ok {
		return fmt.Errorf("unknown resource type %q\nSupported types: svc, pod, ing, node, all", resType)
	}

	// Type only (no name): list all of that type.
	if len(args) == 1 {
		lister, ok := res.(resolver.Lister)
		if !ok {
			return fmt.Errorf("resource type %q does not support listing", resType)
		}
		named, err := lister.ListAll(ctx, client, listNs, domain, selector)
		if err != nil {
			return err
		}
		output.PrintNamedResults(streams, named, allNamespaces, doResolve, outputFormat)
		return nil
	}

	// Type + name: resolve single resource.
	resName := args[1]
	results, err := res.Resolve(ctx, client, ns, resName, domain)
	if err != nil {
		return err
	}
	output.PrintResults(streams, resName, results, doResolve, outputFormat)
	return nil
}

// listAllTypes fetches all resource types concurrently and prints results.
func listAllTypes(ctx context.Context, client kubernetes.Interface, streams genericiooptions.IOStreams, listNs, domain, selector string, withNamespace, doResolve bool, outputFormat string) error {
	var mu sync.Mutex
	combined := make([]resolver.NamedResults, 0)

	g, gctx := errgroup.WithContext(ctx)
	for _, t := range resolver.AllTypes {
		t := t
		g.Go(func() error {
			lister, ok := resolver.Registry[t].(resolver.Lister)
			if !ok {
				return nil
			}
			named, err := lister.ListAll(gctx, client, listNs, domain, selector)
			if err != nil {
				fmt.Fprintf(streams.ErrOut, "warning: listing %s: %v\n", t, err)
				return nil // don't abort the entire operation on a single-type failure
			}
			for i := range named {
				named[i].Type = t
			}
			mu.Lock()
			combined = append(combined, named...)
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	output.PrintNamedResults(streams, combined, withNamespace, doResolve, outputFormat)
	return nil
}
