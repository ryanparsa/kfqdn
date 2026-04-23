# kfqdn

A kubectl plugin that extracts every DNS-relevant name from any Kubernetes resource — services, pods, ingresses, and nodes.

## Project structure

```
kfqdn/
├── cmd/
│   └── main.go
├── internal/
│   ├── cli/
│   │   ├── root.go
│   │   └── run.go
│   ├── resolver/
│   │   ├── resolver.go
│   │   ├── cluster.go
│   │   ├── service.go
│   │   ├── pod.go
│   │   ├── ingress.go
│   │   └── node.go
│   └── output/
│       └── print.go
├── go.mod
├── Makefile
└── README.md
```

## Installation

### From source

```bash
make build
cp kubectl-fqdn ~/.local/bin/
```

That directory must be on your `PATH`:

```bash
export PATH="$PATH:$HOME/.local/bin"
```

### Verify

```bash
kubectl plugin list | grep fqdn
```

## Usage

```
kubectl fqdn <type> [name] [flags]
```

| Type | Aliases | Description |
|------|---------|-------------|
| `svc` | `service`, `services` | Kubernetes Service |
| `pod` | `pods`, `po` | Kubernetes Pod |
| `ing` | `ingress`, `ingresses` | Kubernetes Ingress |
| `node` | `nodes`, `no` | Kubernetes Node |
| `all` | | All resource types |

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--namespace` | `-n` | Namespace scope |
| `--all-namespaces` | `-A` | All namespaces |
| `--resolve` | `-r` | Resolve DNS names to IP addresses |
| `--context` | | Kubeconfig context |
| `--kubeconfig` | | Path to kubeconfig |

## Examples

```bash
# List all services in a namespace
kubectl fqdn svc -n default

# Resolve a specific service
kubectl fqdn svc my-svc -n production

# List all pods in a namespace
kubectl fqdn pod -n kube-system

# Resolve a specific pod
kubectl fqdn pod my-pod-0 -n default

# List all ingresses
kubectl fqdn ing -n prod

# List all nodes
kubectl fqdn node

# All resource types in a namespace
kubectl fqdn all -n kube-system

# All resource types across every namespace
kubectl fqdn all -A

# Resolve DNS names to IPs
kubectl fqdn svc my-svc -n default -r

# List and resolve all services across namespaces
kubectl fqdn svc -A -r
```

### `kubectl fqdn all -A -r`

```
NAMESPACE            TYPE   NAME                                         DNS NAME                                          KIND           IP(S)
default              svc    kubernetes                                   kubernetes.default.svc.cluster.local              cluster-fqdn   192.168.194.129, fd07:b51a:cc66:0:a617:db5e:c0a8:c281
kube-system          svc    kube-dns                                     kube-dns.kube-system.svc.cluster.local            cluster-fqdn   192.168.194.138, fd07:b51a:cc66:a:8000::a
default              pod    dnsutils                                     10-244-0-5.default.pod.cluster.local              ip-pod-fqdn    fd07:b51a:cc66:0:a617:db5e:af4:5, 10.244.0.5
kube-system          pod    coredns-7d764666f9-28r4w                     10-244-0-4.kube-system.pod.cluster.local          ip-pod-fqdn    fd07:b51a:cc66:0:a617:db5e:af4:4, 10.244.0.4
kube-system          pod    coredns-7d764666f9-4jxpl                     10-244-0-2.kube-system.pod.cluster.local          ip-pod-fqdn    fd07:b51a:cc66:0:a617:db5e:af4:2, 10.244.0.2
kube-system          pod    etcd-kind-control-plane                      192-168-97-2.kube-system.pod.cluster.local        ip-pod-fqdn    192.168.97.2, fd07:b51a:cc66:0:a617:db5e:c0a8:6102
kube-system          pod    kindnet-ccrg8                                192-168-97-2.kube-system.pod.cluster.local        ip-pod-fqdn    192.168.97.2, fd07:b51a:cc66:0:a617:db5e:c0a8:6102
kube-system          pod    kube-apiserver-kind-control-plane            192-168-97-2.kube-system.pod.cluster.local        ip-pod-fqdn    192.168.97.2, fd07:b51a:cc66:0:a617:db5e:c0a8:6102
kube-system          pod    kube-controller-manager-kind-control-plane   192-168-97-2.kube-system.pod.cluster.local        ip-pod-fqdn    192.168.97.2, fd07:b51a:cc66:0:a617:db5e:c0a8:6102
kube-system          pod    kube-proxy-6lqpz                             192-168-97-2.kube-system.pod.cluster.local        ip-pod-fqdn    192.168.97.2, fd07:b51a:cc66:0:a617:db5e:c0a8:6102
kube-system          pod    kube-scheduler-kind-control-plane            192-168-97-2.kube-system.pod.cluster.local        ip-pod-fqdn    192.168.97.2, fd07:b51a:cc66:0:a617:db5e:c0a8:6102
local-path-storage   pod    local-path-provisioner-67b8995b4b-l6q98      10-244-0-3.local-path-storage.pod.cluster.local   ip-pod-fqdn    [unresolved]
                     node   kind-control-plane                           kind-control-plane                                hostname       [unresolved]
```

> `[unresolved]` means the DNS name only resolves from inside the cluster, not from your local machine.

## What gets extracted

| Resource | DNS names extracted |
|---|---|
| Service (ClusterIP) | `<name>.<ns>.svc.<domain>` |
| Service (ExternalName) | cluster FQDN + CNAME target |
| Service (LoadBalancer) | cluster FQDN + external hostname/IP |
| Pod (StatefulSet) | `<pod>.<governing-svc>.<ns>.svc.<domain>` |
| Pod (headless service) | `<pod>.<svc>.<ns>.svc.<domain>` |
| Pod (other) | `<ip-dashes>.<ns>.pod.<domain>` |
| Ingress | rule hosts, TLS hosts, LB hostnames |
| Node | external-dns, internal-dns, hostname addresses |

The cluster domain is auto-detected from the CoreDNS ConfigMap in `kube-system`, falling back to `cluster.local`.
