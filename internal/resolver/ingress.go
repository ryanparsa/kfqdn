package resolver

import (
	"context"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type IngressResolver struct{}

func (r *IngressResolver) Resolve(ctx context.Context, client kubernetes.Interface, ns, name, domain string) ([]Result, error) {
	ing, err := client.NetworkingV1().Ingresses(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return resolveIngressObj(ing), nil
}

func (r *IngressResolver) ListAll(ctx context.Context, client kubernetes.Interface, ns, domain string) ([]NamedResults, error) {
	ings, err := client.NetworkingV1().Ingresses(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var named []NamedResults
	for i := range ings.Items {
		ing := &ings.Items[i]
		named = append(named, NamedResults{
			Namespace: ing.Namespace,
			Name:      ing.Name,
			Results:   resolveIngressObj(ing),
		})
	}
	return named, nil
}

func resolveIngressObj(ing *networkingv1.Ingress) []Result {
	seen := map[string]bool{}
	var results []Result

	add := func(name, kind string) {
		if name != "" && !seen[name] {
			seen[name] = true
			results = append(results, Result{Name: name, Kind: kind})
		}
	}

	for _, rule := range ing.Spec.Rules {
		add(rule.Host, "rule-host")
	}
	for _, tls := range ing.Spec.TLS {
		for _, host := range tls.Hosts {
			add(host, "tls-host")
		}
	}
	for _, lbIng := range ing.Status.LoadBalancer.Ingress {
		add(lbIng.Hostname, "lb-hostname")
		add(lbIng.IP, "lb-ip")
	}

	return results
}
