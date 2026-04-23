package resolver

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type ServiceResolver struct{}

func (r *ServiceResolver) Resolve(ctx context.Context, client kubernetes.Interface, ns, name, domain string) ([]Result, error) {
	svc, err := client.CoreV1().Services(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return ServiceResultsFor(svc, domain), nil
}

func (r *ServiceResolver) ListAll(ctx context.Context, client kubernetes.Interface, ns, domain string) ([]NamedResults, error) {
	svcs, err := client.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing services: %w", err)
	}
	named := make([]NamedResults, 0, len(svcs.Items))
	for i := range svcs.Items {
		svc := &svcs.Items[i]
		named = append(named, NamedResults{
			Namespace: svc.Namespace,
			Name:      svc.Name,
			Results:   ServiceResultsFor(svc, domain),
		})
	}
	return named, nil
}

// ServiceResultsFor extracts all DNS-relevant names from an already-fetched Service object.
// Used by both Resolve and output.ListAll (avoids N+1 API calls in list paths).
func ServiceResultsFor(svc *corev1.Service, domain string) []Result {
	clusterFQDN := fmt.Sprintf("%s.%s.svc.%s", svc.Name, svc.Namespace, domain)
	results := []Result{{Name: clusterFQDN, Kind: "cluster-fqdn"}}

	switch svc.Spec.Type {
	case corev1.ServiceTypeExternalName:
		if svc.Spec.ExternalName != "" {
			results = append(results, Result{Name: svc.Spec.ExternalName, Kind: "cname"})
		}
	case corev1.ServiceTypeLoadBalancer:
		for _, ing := range svc.Status.LoadBalancer.Ingress {
			if ing.Hostname != "" {
				results = append(results, Result{Name: ing.Hostname, Kind: "external-hostname"})
			}
			if ing.IP != "" {
				results = append(results, Result{Name: ing.IP, Kind: "external-ip"})
			}
		}
	}

	return results
}
