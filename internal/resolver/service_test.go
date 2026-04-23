package resolver_test

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/imryanparsa/kfqdn/internal/resolver"
)

const testDomain = "cluster.local"

func makeService(ns, name string, svcType corev1.ServiceType, ports []corev1.ServicePort, extra ...func(*corev1.Service)) *corev1.Service {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: corev1.ServiceSpec{
			Type:  svcType,
			Ports: ports,
		},
	}
	for _, fn := range extra {
		fn(svc)
	}
	return svc
}

func TestServiceResolver_ClusterIP(t *testing.T) {
	svc := makeService("default", "my-svc", corev1.ServiceTypeClusterIP,
		[]corev1.ServicePort{{Port: 80, Protocol: corev1.ProtocolTCP}})
	client := fake.NewSimpleClientset(svc)

	r := &resolver.ServiceResolver{}
	results, err := r.Resolve(context.Background(), client, "default", "my-svc", testDomain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	want := "my-svc.default.svc.cluster.local"
	if results[0].Name != want {
		t.Errorf("expected FQDN %q, got %q", want, results[0].Name)
	}
	if results[0].Kind != "cluster-fqdn" {
		t.Errorf("expected kind cluster-fqdn, got %q", results[0].Kind)
	}
}

func TestServiceResolver_ExternalName(t *testing.T) {
	svc := makeService("default", "ext-svc", corev1.ServiceTypeExternalName, nil,
		func(s *corev1.Service) { s.Spec.ExternalName = "example.com" })
	client := fake.NewSimpleClientset(svc)

	r := &resolver.ServiceResolver{}
	results, err := r.Resolve(context.Background(), client, "default", "ext-svc", testDomain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Expect cluster-fqdn + cname
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[1].Kind != "cname" || results[1].Name != "example.com" {
		t.Errorf("unexpected cname result: %+v", results[1])
	}
}

func TestServiceResolver_LoadBalancer(t *testing.T) {
	svc := makeService("default", "lb-svc", corev1.ServiceTypeLoadBalancer, nil,
		func(s *corev1.Service) {
			s.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{
				{Hostname: "lb.example.com"},
				{IP: "1.2.3.4"},
			}
		})
	client := fake.NewSimpleClientset(svc)

	r := &resolver.ServiceResolver{}
	results, err := r.Resolve(context.Background(), client, "default", "lb-svc", testDomain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// cluster-fqdn + external-hostname + external-ip
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d: %+v", len(results), results)
	}
	if results[1].Kind != "external-hostname" {
		t.Errorf("expected external-hostname, got %q", results[1].Kind)
	}
	if results[2].Kind != "external-ip" {
		t.Errorf("expected external-ip, got %q", results[2].Kind)
	}
}

func TestServiceResolver_ListAll_LabelSelector(t *testing.T) {
	svc1 := makeService("default", "svc1", corev1.ServiceTypeClusterIP, nil,
		func(s *corev1.Service) { s.Labels = map[string]string{"app": "a"} })
	svc2 := makeService("default", "svc2", corev1.ServiceTypeClusterIP, nil,
		func(s *corev1.Service) { s.Labels = map[string]string{"app": "b"} })
	client := fake.NewSimpleClientset(svc1, svc2)

	r := &resolver.ServiceResolver{}
	// Without selector: should return both.
	named, err := r.ListAll(context.Background(), client, "default", testDomain, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(named) != 2 {
		t.Errorf("expected 2 named results, got %d", len(named))
	}
}

func TestServiceResolver_ListAll_Ports(t *testing.T) {
	svc := makeService("default", "portsvc", corev1.ServiceTypeClusterIP,
		[]corev1.ServicePort{
			{Port: 80, Protocol: corev1.ProtocolTCP},
			{Port: 443, Protocol: corev1.ProtocolTCP},
		})
	client := fake.NewSimpleClientset(svc)

	r := &resolver.ServiceResolver{}
	named, err := r.ListAll(context.Background(), client, "default", testDomain, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(named) != 1 {
		t.Fatalf("expected 1 named result, got %d", len(named))
	}
	if named[0].Extra != "80/TCP,443/TCP" {
		t.Errorf("expected Extra=80/TCP,443/TCP, got %q", named[0].Extra)
	}
}

func TestServiceResolver_NotFound(t *testing.T) {
	client := fake.NewSimpleClientset()

	r := &resolver.ServiceResolver{}
	_, err := r.Resolve(context.Background(), client, "default", "missing", testDomain)
	if err == nil {
		t.Fatal("expected error for missing service, got nil")
	}
}
