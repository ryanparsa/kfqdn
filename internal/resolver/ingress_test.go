package resolver_test

import (
	"context"
	"testing"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/imryanparsa/kfqdn/internal/resolver"
)

func makeIngress(ns, name string, ruleHosts, tlsHosts []string, lbIngress []networkingv1.IngressLoadBalancerIngress) *networkingv1.Ingress {
	var rules []networkingv1.IngressRule
	for _, h := range ruleHosts {
		rules = append(rules, networkingv1.IngressRule{Host: h})
	}
	var tls []networkingv1.IngressTLS
	if len(tlsHosts) > 0 {
		tls = []networkingv1.IngressTLS{{Hosts: tlsHosts}}
	}
	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: networkingv1.IngressSpec{
			Rules: rules,
			TLS:   tls,
		},
		Status: networkingv1.IngressStatus{
			LoadBalancer: networkingv1.IngressLoadBalancerStatus{Ingress: lbIngress},
		},
	}
}

func TestIngressResolver_RuleHosts(t *testing.T) {
	ing := makeIngress("default", "my-ing", []string{"foo.example.com", "bar.example.com"}, nil, nil)
	client := fake.NewSimpleClientset(ing)

	r := &resolver.IngressResolver{}
	results, err := r.Resolve(context.Background(), client, "default", "my-ing", testDomain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Kind != "rule-host" {
			t.Errorf("expected rule-host, got %q", r.Kind)
		}
	}
}

func TestIngressResolver_TLSHosts(t *testing.T) {
	// A host that appears in both rules and TLS should only appear once for rule-host.
	ing := makeIngress("default", "tls-ing",
		[]string{"secure.example.com"},
		[]string{"secure.example.com", "extra.example.com"},
		nil,
	)
	client := fake.NewSimpleClientset(ing)

	r := &resolver.IngressResolver{}
	results, err := r.Resolve(context.Background(), client, "default", "tls-ing", testDomain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// rule-host for "secure.example.com", tls-host for "extra.example.com"
	// "secure.example.com" is already added as rule-host so it won't be re-added as tls-host.
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d: %+v", len(results), results)
	}
}

func TestIngressResolver_LBStatus(t *testing.T) {
	ing := makeIngress("default", "lb-ing", nil, nil, []networkingv1.IngressLoadBalancerIngress{
		{Hostname: "lb.example.com"},
		{IP: "5.6.7.8"},
	})
	client := fake.NewSimpleClientset(ing)

	r := &resolver.IngressResolver{}
	results, err := r.Resolve(context.Background(), client, "default", "lb-ing", testDomain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results (lb-hostname + lb-ip), got %d", len(results))
	}
	kinds := map[string]bool{}
	for _, r := range results {
		kinds[r.Kind] = true
	}
	if !kinds["lb-hostname"] || !kinds["lb-ip"] {
		t.Errorf("expected lb-hostname and lb-ip kinds, got %v", kinds)
	}
}

func TestIngressResolver_ListAll(t *testing.T) {
	ing1 := makeIngress("default", "ing1", []string{"a.example.com"}, nil, nil)
	ing2 := makeIngress("default", "ing2", []string{"b.example.com"}, nil, nil)
	client := fake.NewSimpleClientset(ing1, ing2)

	r := &resolver.IngressResolver{}
	named, err := r.ListAll(context.Background(), client, "default", testDomain, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(named) != 2 {
		t.Fatalf("expected 2 named results, got %d", len(named))
	}
}

func TestIngressResolver_Empty(t *testing.T) {
	// An ingress with no hosts/TLS/LB should produce an empty results slice.
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "empty", Namespace: "default"},
	}
	client := fake.NewSimpleClientset(ing)

	r := &resolver.IngressResolver{}
	results, err := r.Resolve(context.Background(), client, "default", "empty", testDomain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty ingress, got %d", len(results))
	}
}

func TestIngressResolver_NotFound(t *testing.T) {
	client := fake.NewSimpleClientset()
	r := &resolver.IngressResolver{}
	_, err := r.Resolve(context.Background(), client, "default", "missing", testDomain)
	if err == nil {
		t.Fatal("expected error for missing ingress, got nil")
	}
}
