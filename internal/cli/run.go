package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/imryanparsa/kfqdn/internal/output"
	"github.com/imryanparsa/kfqdn/internal/resolver"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/client-go/kubernetes"
)

func run(configFlags *genericclioptions.ConfigFlags, streams genericiooptions.IOStreams, args []string, allNamespaces, doResolve bool) error {
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

	ctx := context.Background()
	domain := resolver.ClusterDomain(ctx, client)

	// No type given + -A: list all services (original behavior).
	if len(args) == 0 {
		return output.ListAll(ctx, client, streams, domain, doResolve)
	}

	resType := strings.ToLower(args[0])

	// "all" aggregates every resource type with a TYPE column.
	if resType == "all" && len(args) == 1 {
		listNs := ns
		if allNamespaces {
			listNs = ""
		}
		var combined []resolver.NamedResults
		for _, t := range resolver.AllTypes {
			lister, ok := resolver.Registry[t].(resolver.Lister)
			if !ok {
				continue
			}
			named, err := lister.ListAll(ctx, client, listNs, domain)
			if err != nil {
				continue
			}
			for i := range named {
				named[i].Type = t
			}
			combined = append(combined, named...)
		}
		output.PrintNamedResults(streams, combined, allNamespaces, doResolve)
		return nil
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
		listNs := ns
		if allNamespaces {
			listNs = ""
		}
		named, err := lister.ListAll(ctx, client, listNs, domain)
		if err != nil {
			return err
		}
		output.PrintNamedResults(streams, named, allNamespaces, doResolve)
		return nil
	}

	// Type + name: resolve single resource.
	resName := args[1]
	results, err := res.Resolve(ctx, client, ns, resName, domain)
	if err != nil {
		return err
	}
	output.PrintResults(streams, resName, results, doResolve)
	return nil
}
