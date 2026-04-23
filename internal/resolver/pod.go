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
		return nil, fmt.Errorf("pod %q not found in namespace %q", name, ns)
	}
	return r.resolveFromObj(ctx, client, pod, domain)
}

func (r *PodResolver) ListAll(ctx context.Context, client kubernetes.Interface, ns, domain string) ([]NamedResults, error) {
	pods, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing pods: %w", err)
	}
	var named []NamedResults
	for i := range pods.Items {
		results, err := r.resolveFromObj(ctx, client, &pods.Items[i], domain)
		if err != nil {
			continue
		}
		named = append(named, NamedResults{
			Namespace: pods.Items[i].Namespace,
			Name:      pods.Items[i].Name,
			Results:   results,
		})
	}
	return named, nil
}

func (r *PodResolver) resolveFromObj(ctx context.Context, client kubernetes.Interface, pod *corev1.Pod, domain string) ([]Result, error) {
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

	if svcName, ok := headlessSvcForPod(ctx, client, ns, pod.Labels); ok {
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

func headlessSvcForPod(ctx context.Context, client kubernetes.Interface, ns string, podLabels map[string]string) (string, bool) {
	svcs, err := client.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", false
	}
	for _, svc := range svcs.Items {
		if svc.Spec.ClusterIP != "None" || len(svc.Spec.Selector) == 0 {
			continue
		}
		if labels.SelectorFromSet(labels.Set(svc.Spec.Selector)).Matches(labels.Set(podLabels)) {
			return svc.Name, true
		}
	}
	return "", false
}
