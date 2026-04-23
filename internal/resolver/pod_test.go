package resolver_test

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/imryanparsa/kfqdn/internal/resolver"
)

func makePod(ns, name, ip string, labels map[string]string, owners ...metav1.OwnerReference) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       ns,
			Labels:          labels,
			OwnerReferences: owners,
		},
		Status: corev1.PodStatus{
			PodIP: ip,
			Phase: corev1.PodRunning,
		},
	}
}

func makeStatefulSet(ns, name, svcName string) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec:       appsv1.StatefulSetSpec{ServiceName: svcName},
	}
}

func makeHeadlessSvc(ns, name string, selector map[string]string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Selector:  selector,
		},
	}
}

func TestPodResolver_StatefulSet(t *testing.T) {
	sts := makeStatefulSet("default", "my-sts", "my-headless")
	pod := makePod("default", "my-sts-0", "10.0.0.1", nil,
		metav1.OwnerReference{Kind: "StatefulSet", Name: "my-sts"})
	client := fake.NewSimpleClientset(sts, pod)

	r := &resolver.PodResolver{}
	results, err := r.Resolve(context.Background(), client, "default", "my-sts-0", testDomain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	want := "my-sts-0.my-headless.default.svc.cluster.local"
	if results[0].Name != want {
		t.Errorf("expected %q, got %q", want, results[0].Name)
	}
	if results[0].Kind != "statefulset-fqdn" {
		t.Errorf("expected statefulset-fqdn, got %q", results[0].Kind)
	}
}

func TestPodResolver_Headless(t *testing.T) {
	labels := map[string]string{"app": "myapp"}
	pod := makePod("default", "my-pod", "10.0.0.2", labels)
	svc := makeHeadlessSvc("default", "headless-svc", labels)
	client := fake.NewSimpleClientset(pod, svc)

	r := &resolver.PodResolver{}
	results, err := r.Resolve(context.Background(), client, "default", "my-pod", testDomain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	want := "my-pod.headless-svc.default.svc.cluster.local"
	if results[0].Name != want {
		t.Errorf("expected %q, got %q", want, results[0].Name)
	}
	if results[0].Kind != "headless-fqdn" {
		t.Errorf("expected headless-fqdn, got %q", results[0].Kind)
	}
}

func TestPodResolver_IPBased(t *testing.T) {
	pod := makePod("default", "my-pod", "10.244.0.5", nil)
	client := fake.NewSimpleClientset(pod)

	r := &resolver.PodResolver{}
	results, err := r.Resolve(context.Background(), client, "default", "my-pod", testDomain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "10-244-0-5.default.pod.cluster.local"
	if results[0].Name != want {
		t.Errorf("expected %q, got %q", want, results[0].Name)
	}
	if results[0].Kind != "ip-pod-fqdn" {
		t.Errorf("expected ip-pod-fqdn, got %q", results[0].Kind)
	}
}

func TestPodResolver_NoIP_Resolve(t *testing.T) {
	pod := makePod("default", "no-ip-pod", "", nil)
	client := fake.NewSimpleClientset(pod)

	r := &resolver.PodResolver{}
	_, err := r.Resolve(context.Background(), client, "default", "no-ip-pod", testDomain)
	if err == nil {
		t.Fatal("expected error for pod with no IP, got nil")
	}
}

func TestPodResolver_NoIP_ListAll_Placeholder(t *testing.T) {
	pod := makePod("default", "no-ip-pod", "", nil)
	client := fake.NewSimpleClientset(pod)

	r := &resolver.PodResolver{}
	named, err := r.ListAll(context.Background(), client, "default", testDomain, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(named) != 1 {
		t.Fatalf("expected 1 named result (placeholder), got %d", len(named))
	}
	if named[0].Results[0].Name != "[no ip]" {
		t.Errorf("expected placeholder [no ip], got %q", named[0].Results[0].Name)
	}
}

func TestPodResolver_ListAll_NoPlusOneRequests(t *testing.T) {
	// Three pods, one headless service — services must be fetched only once.
	labels := map[string]string{"app": "app"}
	svc := makeHeadlessSvc("default", "headless", labels)
	pods := []corev1.Pod{
		*makePod("default", "pod-0", "10.0.0.1", labels),
		*makePod("default", "pod-1", "10.0.0.2", labels),
		*makePod("default", "pod-2", "10.0.0.3", labels),
	}
	objs := []interface{}{svc}
	for i := range pods {
		objs = append(objs, &pods[i])
	}
	// fake.NewSimpleClientset accepts runtime.Object
	client := fake.NewSimpleClientset(svc, &pods[0], &pods[1], &pods[2])

	r := &resolver.PodResolver{}
	named, err := r.ListAll(context.Background(), client, "default", testDomain, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(named) != 3 {
		t.Fatalf("expected 3 named results, got %d", len(named))
	}
	for _, nr := range named {
		if nr.Results[0].Kind != "headless-fqdn" {
			t.Errorf("expected headless-fqdn for %q, got %q", nr.Name, nr.Results[0].Kind)
		}
	}
}

func TestPodResolver_ListAll_Phase(t *testing.T) {
	pod := makePod("default", "running-pod", "10.0.0.1", nil)
	pod.Status.Phase = corev1.PodRunning
	client := fake.NewSimpleClientset(pod)

	r := &resolver.PodResolver{}
	named, err := r.ListAll(context.Background(), client, "default", testDomain, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if named[0].Extra != "Running" {
		t.Errorf("expected Extra=Running, got %q", named[0].Extra)
	}
}

func TestPodResolver_NotFound(t *testing.T) {
	client := fake.NewSimpleClientset()

	r := &resolver.PodResolver{}
	_, err := r.Resolve(context.Background(), client, "default", "missing", testDomain)
	if err == nil {
		t.Fatal("expected error for missing pod, got nil")
	}
}
