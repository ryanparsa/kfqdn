package resolver_test

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/imryanparsa/kfqdn/internal/resolver"
)

func TestClusterDomain_FromConfigMap(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "coredns",
			Namespace: "kube-system",
		},
		Data: map[string]string{
			"Corefile": `.:53 {
    kubernetes mycluster.local in-addr.arpa ip6.arpa {
        pods insecure
        fallthrough in-addr.arpa ip6.arpa
    }
}`,
		},
	}
	client := fake.NewSimpleClientset(cm)

	domain := resolver.ClusterDomain(context.Background(), client)
	if domain != "mycluster.local" {
		t.Errorf("expected mycluster.local, got %q", domain)
	}
}

func TestClusterDomain_DefaultFallback(t *testing.T) {
	// No ConfigMap → should fall back to cluster.local.
	client := fake.NewSimpleClientset()

	domain := resolver.ClusterDomain(context.Background(), client)
	if domain != "cluster.local" {
		t.Errorf("expected cluster.local fallback, got %q", domain)
	}
}

func TestClusterDomain_MalformedCorefile(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "coredns",
			Namespace: "kube-system",
		},
		Data: map[string]string{
			"Corefile": `.:53 { health }`, // no kubernetes directive
		},
	}
	client := fake.NewSimpleClientset(cm)

	domain := resolver.ClusterDomain(context.Background(), client)
	if domain != "cluster.local" {
		t.Errorf("expected cluster.local fallback for malformed Corefile, got %q", domain)
	}
}

func TestClusterDomain_TrailingDot(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "coredns",
			Namespace: "kube-system",
		},
		Data: map[string]string{
			"Corefile": `kubernetes cluster.local. in-addr.arpa`,
		},
	}
	client := fake.NewSimpleClientset(cm)

	domain := resolver.ClusterDomain(context.Background(), client)
	if domain != "cluster.local" {
		t.Errorf("expected trailing dot stripped, got %q", domain)
	}
}
