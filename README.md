<div align="center">

# kubectl-fqdn

**Extract every DNS-relevant name from your Kubernetes cluster — in one command.**

[![CI](https://github.com/imryanparsa/kfqdn/actions/workflows/ci.yml/badge.svg)](https://github.com/imryanparsa/kfqdn/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/imryanparsa/kfqdn)](https://go.dev/doc/devel/release)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

</div>

---

Debugging DNS in Kubernetes usually means jumping between `kubectl get svc`, `kubectl get pods`, and manually constructing FQDNs. `kubectl fqdn` does it for you — for services, pods, ingresses, and nodes — with optional live DNS resolution.

## Features

- Auto-detects your cluster domain from the CoreDNS ConfigMap (falls back to `cluster.local`)
- Handles all service types: ClusterIP, ExternalName, LoadBalancer
- Resolves pod FQDNs: StatefulSet governing service, headless service, and IP-based
- Extracts ingress rule hosts, TLS hosts, and LB hostnames/IPs
- Resolves node external/internal DNS and hostnames
- Optional live DNS resolution with `--resolve` / `-r`
- Inherits all standard kubectl flags (`--namespace`, `--context`, `--kubeconfig`, etc.)

## Installation

### Via Krew

```bash
kubectl krew install fqdn
```

### Build from source

```bash
git clone https://github.com/imryanparsa/kfqdn.git
cd kfqdn
make install
```

> **Note:** The install target builds the binary and prints copy instructions. Ensure the target directory is on your `PATH`.

### Verify

```bash
kubectl plugin list | grep fqdn
```

## Usage

```
kubectl fqdn <type> [name] [flags]
```

### Examples

```bash
# All services in the current namespace
kubectl fqdn svc

# One specific service
kubectl fqdn svc my-service -n production

# All resource types across every namespace
kubectl fqdn all -A

# Resolve DNS names to live IPs
kubectl fqdn svc -n default --resolve

# Use a non-default context
kubectl fqdn pod --context staging-cluster -n app
```

## Supported Resource Types

| Type | Aliases | What gets extracted |
|------|---------|---------------------|
| `svc` | `service`, `services` | Cluster FQDN · CNAME target (ExternalName) · external hostname/IP (LoadBalancer) |
| `pod` | `pods`, `po` | StatefulSet FQDN · headless service FQDN · IP-based FQDN |
| `ing` | `ingress`, `ingresses` | Rule hosts · TLS hosts · LB hostnames/IPs |
| `node` | `nodes`, `no` | External DNS · internal DNS · hostname |
| `all` | | All of the above |

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--namespace` | `-n` | current context | Target namespace |
| `--all-namespaces` | `-A` | `false` | Query across all namespaces |
| `--resolve` | `-r` | `false` | Resolve DNS names to IP addresses |
| `--context` | | current context | Kubeconfig context to use |
| `--kubeconfig` | | `~/.kube/config` | Path to kubeconfig file |

## Example Output

```
kubectl fqdn all -A -r
```

```
NAMESPACE            TYPE   NAME                                         DNS NAME                                          KIND             IP(S)
default              svc    kubernetes                                   kubernetes.default.svc.cluster.local              cluster-fqdn     192.168.194.129, fd07:b51a:cc66:0:a617:db5e:c0a8:c281
kube-system          svc    kube-dns                                     kube-dns.kube-system.svc.cluster.local            cluster-fqdn     192.168.194.138, fd07:b51a:cc66:a:8000::a
default              pod    dnsutils                                     10-244-0-5.default.pod.cluster.local              ip-pod-fqdn      10.244.0.5
kube-system          pod    coredns-7d764666f9-28r4w                     10-244-0-4.kube-system.pod.cluster.local          ip-pod-fqdn      10.244.0.4
kube-system          pod    etcd-kind-control-plane                      192-168-97-2.kube-system.pod.cluster.local        ip-pod-fqdn      192.168.97.2
local-path-storage   pod    local-path-provisioner-67b8995b4b-l6q98      10-244-0-3.local-path-storage.pod.cluster.local   ip-pod-fqdn      [unresolved]
                     node   kind-control-plane                           kind-control-plane                                hostname         [unresolved]
```

> `[unresolved]` means the name resolves only from inside the cluster — this is expected when running from your local machine.

## How It Works

On startup, `kubectl fqdn` reads the `kube-system/coredns` ConfigMap and extracts the cluster domain from the Corefile (`kubernetes <domain>` directive). If detection fails it falls back to `cluster.local`. All FQDNs are then constructed according to the [Kubernetes DNS specification](https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for how to set up your environment, add a new resource type, and open a pull request.

## License

MIT — see [LICENSE](LICENSE) for details.
