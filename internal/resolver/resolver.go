package resolver

import (
	"context"

	"k8s.io/client-go/kubernetes"
)

// Result is a single DNS-relevant name extracted from a Kubernetes resource.
type Result struct {
	Name string // DNS name, hostname, IP, or CNAME target
	Kind string // "cluster-fqdn", "cname", "external-hostname", "lb-hostname", "rule-host", "tls-host", "statefulset-fqdn", "headless-fqdn", "ip-pod-fqdn", "external-dns", "internal-dns", "hostname"
}

// Resolver extracts all DNS-relevant names from a named Kubernetes resource.
type Resolver interface {
	Resolve(ctx context.Context, client kubernetes.Interface, ns, name, domain string) ([]Result, error)
}

// NamedResults groups a resource's identity with its DNS results.
type NamedResults struct {
	Namespace string
	Name      string
	Type      string // set when multiple resource types are mixed (e.g. "all")
	Results   []Result
}

// AllTypes is the canonical ordered list used by "all" to iterate every Lister.
var AllTypes = []string{"svc", "pod", "ing", "node"}

// Lister can enumerate all resources of a type within a namespace.
// An empty ns string means all namespaces.
type Lister interface {
	ListAll(ctx context.Context, client kubernetes.Interface, ns, domain string) ([]NamedResults, error)
}

// Registry maps resource type aliases to their Resolver.
var Registry = map[string]Resolver{
	"svc":      &ServiceResolver{},
	"service":  &ServiceResolver{},
	"services": &ServiceResolver{},
	"pod":      &PodResolver{},
	"pods":     &PodResolver{},
	"po":       &PodResolver{},
	"ing":      &IngressResolver{},
	"ingress":  &IngressResolver{},
	"ingresses": &IngressResolver{},
	"node":     &NodeResolver{},
	"nodes":    &NodeResolver{},
	"no":       &NodeResolver{},
}
