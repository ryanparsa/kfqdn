package resolver_test

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/imryanparsa/kfqdn/internal/resolver"
)

func makeNode(name string, addrs []corev1.NodeAddress) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status:     corev1.NodeStatus{Addresses: addrs},
	}
}

func TestNodeResolver_AllAddressTypes(t *testing.T) {
	node := makeNode("node-1", []corev1.NodeAddress{
		{Type: corev1.NodeExternalDNS, Address: "external.example.com"},
		{Type: corev1.NodeInternalDNS, Address: "internal.example.com"},
		{Type: corev1.NodeHostName, Address: "node-1"},
	})
	client := fake.NewSimpleClientset(node)

	r := &resolver.NodeResolver{}
	results, err := r.Resolve(context.Background(), client, "", "node-1", testDomain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d: %+v", len(results), results)
	}
	kinds := map[string]string{}
	for _, r := range results {
		kinds[r.Kind] = r.Name
	}
	if kinds["external-dns"] != "external.example.com" {
		t.Errorf("unexpected external-dns: %q", kinds["external-dns"])
	}
	if kinds["internal-dns"] != "internal.example.com" {
		t.Errorf("unexpected internal-dns: %q", kinds["internal-dns"])
	}
	if kinds["hostname"] != "node-1" {
		t.Errorf("unexpected hostname: %q", kinds["hostname"])
	}
}

func TestNodeResolver_OnlyHostname(t *testing.T) {
	node := makeNode("worker-1", []corev1.NodeAddress{
		{Type: corev1.NodeHostName, Address: "worker-1"},
	})
	client := fake.NewSimpleClientset(node)

	r := &resolver.NodeResolver{}
	results, err := r.Resolve(context.Background(), client, "", "worker-1", testDomain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].Kind != "hostname" {
		t.Errorf("expected 1 hostname result, got %+v", results)
	}
}

func TestNodeResolver_ListAll(t *testing.T) {
	n1 := makeNode("node-a", []corev1.NodeAddress{{Type: corev1.NodeHostName, Address: "node-a"}})
	n2 := makeNode("node-b", []corev1.NodeAddress{{Type: corev1.NodeHostName, Address: "node-b"}})
	client := fake.NewSimpleClientset(n1, n2)

	r := &resolver.NodeResolver{}
	named, err := r.ListAll(context.Background(), client, "", testDomain, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(named) != 2 {
		t.Fatalf("expected 2 named results, got %d", len(named))
	}
}

func TestNodeResolver_IgnoresExternalIP(t *testing.T) {
	// NodeExternalIP is not a DNS type; it should be excluded.
	node := makeNode("node-1", []corev1.NodeAddress{
		{Type: corev1.NodeExternalIP, Address: "203.0.113.1"},
		{Type: corev1.NodeHostName, Address: "node-1"},
	})
	client := fake.NewSimpleClientset(node)

	r := &resolver.NodeResolver{}
	results, err := r.Resolve(context.Background(), client, "", "node-1", testDomain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only hostname should be returned; ExternalIP is not a DNS-relevant type.
	if len(results) != 1 || results[0].Kind != "hostname" {
		t.Errorf("expected only hostname result, got %+v", results)
	}
}

func TestNodeResolver_NotFound(t *testing.T) {
	client := fake.NewSimpleClientset()
	r := &resolver.NodeResolver{}
	_, err := r.Resolve(context.Background(), client, "", "missing", testDomain)
	if err == nil {
		t.Fatal("expected error for missing node, got nil")
	}
}
