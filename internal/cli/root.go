package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/client-go/kubernetes"
)

// validTypes is the ordered list of resource type names exposed for shell completion.
var validTypes = []string{"svc", "pod", "ing", "node", "all"}

// NewRootCmd builds the root cobra command with all flags registered.
func NewRootCmd() *cobra.Command {
	configFlags := genericclioptions.NewConfigFlags(true)
	streams := genericiooptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}

	var allNamespaces bool
	var doResolve bool
	var timeout string
	var outputFormat string
	var selector string

	cmd := &cobra.Command{
		Use:   "kfqdn [type name]",
		Short: "Print every DNS name for any Kubernetes resource",
		Long: `Extracts every DNS-relevant name from any Kubernetes resource.

Supported types: svc, pod, ing, node, all

Examples:
  kubectl fqdn svc -n default              list all services in namespace
  kubectl fqdn svc my-svc -n production    resolve a specific service
  kubectl fqdn pod -n kube-system          list all pods in namespace
  kubectl fqdn pod my-pod-0 -n default     resolve a specific pod
  kubectl fqdn ing -n prod                 list all ingresses
  kubectl fqdn node                        list all nodes
  kubectl fqdn all -n kube-system          all resource types in namespace
  kubectl fqdn all -A                      all resource types across namespaces
  kubectl fqdn svc -A                      list all services across namespaces
  kubectl fqdn svc my-svc -r              resolve to IP address(es)`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !allNamespaces {
				return fmt.Errorf("resource type required\n\nUsage:\n  kubectl fqdn <type> [name] [flags]\n\nSupported types: svc, pod, ing, node, all\n\nExamples:\n  kubectl fqdn svc -n default\n  kubectl fqdn all -n kube-system")
			}
			if len(args) > 2 {
				return fmt.Errorf("too many arguments: expected \"<type> [name]\"")
			}
			return nil
		},
		ValidArgs: validTypes,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return validTypes, cobra.ShellCompDirectiveNoFileComp
			}
			// Second argument is a resource name — no static completion available.
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE:         func(cmd *cobra.Command, args []string) error { return run(configFlags, streams, args, allNamespaces, doResolve, timeout, outputFormat, selector) },
		SilenceUsage: true,
	}

	// Register namespace completion for the -n flag.
	_ = cmd.RegisterFlagCompletionFunc("namespace", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		restConfig, err := configFlags.ToRESTConfig()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		client, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		nsList, err := client.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		names := make([]string, 0, len(nsList.Items))
		for _, ns := range nsList.Items {
			names = append(names, ns.Name)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	})

	configFlags.AddFlags(cmd.Flags())
	cmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "List DNS names for all services across all namespaces")
	cmd.Flags().BoolVarP(&doResolve, "resolve", "r", false, "Resolve each DNS name to its IP address(es)")
	cmd.Flags().StringVar(&timeout, "timeout", "30s", "Timeout for Kubernetes API requests (e.g. 30s, 1m)")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format: table, wide, json")
	cmd.Flags().StringVarP(&selector, "selector", "l", "", "Label selector to filter resources (e.g. app=myapp)")

	cmd.AddCommand(newVersionCmd())

	return cmd
}

