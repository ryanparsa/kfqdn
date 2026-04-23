package resolver

import (
	"context"
	"regexp"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ClusterDomain reads the cluster domain from the CoreDNS ConfigMap.
// Falls back to "cluster.local" if it cannot be determined.
func ClusterDomain(ctx context.Context, client kubernetes.Interface) string {
	cm, err := client.CoreV1().ConfigMaps("kube-system").Get(ctx, "coredns", metav1.GetOptions{})
	if err != nil {
		return "cluster.local"
	}
	re := regexp.MustCompile(`kubernetes\s+(\S+)`)
	if m := re.FindStringSubmatch(cm.Data["Corefile"]); len(m) >= 2 {
		return strings.TrimSuffix(m[1], ".")
	}
	return "cluster.local"
}
