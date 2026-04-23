package resolver

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type PodResolver struct{}

func (r *PodResolver) Resolve(ctx context.Context, client kubernetes.Interface, ns, name, domain string) ([]Result, error) {
	pod, err := client.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting pod %q in namespace %q: %w", name, ns, err)
	}
	svcs, err := client.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing services in namespace %q: %w", ns, err)
	}
	return r.resolveFromObj(ctx, client, pod, svcs.Items, domain)
}

func (r *PodResolver) ListAll(ctx context.Context, client kubernetes.Interface, ns, domain, selector string) ([]NamedResults, error) {
	pods, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, fmt.Errorf("listing pods: %w", err)
	}

	// Fetch services once to avoid N+1 calls when matching headless services.
	svcs, err := client.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing services for pod resolution: %w", err)
	}

	var named []NamedResults
	for i := range pods.Items {
		pod := &pods.Items[i]
		results, err := r.resolveFromObj(ctx, client, pod, svcs.Items, domain)
		if err != nil {
			// Emit a placeholder row so the pod is still visible in the output.
			results = []Result{{Name: "[no ip]", Kind: "ip-pod-fqdn"}}
		}
		named = append(named, NamedResults{
			Namespace: pod.Namespace,
			Name:      pod.Name,
			Results:   results,
			Extra:     string(pod.Status.Phase),
		})
	}
	return named, nil
}

func (r *PodResolver) resolveFromObj(ctx context.Context, client kubernetes.Interface, pod *corev1.Pod, svcs []corev1.Service, domain string) ([]Result, error) {
	ns := pod.Namespace
	name := pod.Name

	for _, ref := range pod.OwnerReferences {
		if ref.Kind == "StatefulSet" {
			sts, err := client.AppsV1().StatefulSets(ns).Get(ctx, ref.Name, metav1.GetOptions{})
			if err == nil && sts.Spec.ServiceName != "" {
				return []Result{{
					Name: fmt.Sprintf("%s.%s.%s.svc.%s", name, sts.Spec.ServiceName, ns, domain),
					Kind: "statefulset-fqdn",
				}}, nil
			}
		}
	}

	if svcName, ok := headlessSvcForPod(svcs, pod.Labels); ok {
		return []Result{{
			Name: fmt.Sprintf("%s.%s.%s.svc.%s", name, svcName, ns, domain),
			Kind: "headless-fqdn",
		}}, nil
	}

	if pod.Status.PodIP == "" {
		return nil, fmt.Errorf("pod %q has no IP assigned yet", name)
	}
	dashes := strings.NewReplacer(".", "-", ":", "-").Replace(pod.Status.PodIP)
	return []Result{{
		Name: fmt.Sprintf("%s.%s.pod.%s", dashes, ns, domain),
		Kind: "ip-pod-fqdn",
	}}, nil
}

// headlessSvcForPod matches pod labels against the in-memory service list,
// eliminating the need for an API call per pod.
func headlessSvcForPod(svcs []corev1.Service, podLabels map[string]string) (string, bool) {
	for _, svc := range svcs {
		if svc.Spec.ClusterIP != "None" || len(svc.Spec.Selector) == 0 {
			continue
		}
		if labels.SelectorFromSet(labels.Set(svc.Spec.Selector)).Matches(labels.Set(podLabels)) {
			return svc.Name, true
		}
	}
	return "", false
}
