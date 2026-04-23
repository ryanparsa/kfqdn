package resolver

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type NodeResolver struct{}

func (r *NodeResolver) Resolve(ctx context.Context, client kubernetes.Interface, ns, name, domain string) ([]Result, error) {
	node, err := client.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return resolveNodeObj(node), nil
}

func (r *NodeResolver) ListAll(ctx context.Context, client kubernetes.Interface, ns, domain, selector string) ([]NamedResults, error) {
	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, err
	}
	var named []NamedResults
	for i := range nodes.Items {
		node := &nodes.Items[i]
		named = append(named, NamedResults{
			Namespace: "",
			Name:      node.Name,
			Results:   resolveNodeObj(node),
		})
	}
	return named, nil
}

func resolveNodeObj(node *corev1.Node) []Result {
	var results []Result
	for _, addr := range node.Status.Addresses {
		switch addr.Type {
		case corev1.NodeExternalDNS:
			results = append(results, Result{Name: addr.Address, Kind: "external-dns"})
		case corev1.NodeInternalDNS:
			results = append(results, Result{Name: addr.Address, Kind: "internal-dns"})
		case corev1.NodeHostName:
			results = append(results, Result{Name: addr.Address, Kind: "hostname"})
		}
	}
	return results
}
