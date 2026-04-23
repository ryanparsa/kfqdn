# Contributing to kubectl-fqdn

Thank you for your interest in contributing! This document covers everything you need to go from zero to a merged pull request.

## Prerequisites

- Go 1.23+
- `kubectl` and a running Kubernetes cluster (for manual testing)
- `make`

## Getting Started

```bash
git clone https://github.com/imryanparsa/kfqdn.git
cd kfqdn
go mod download
make build
```

This produces a `kubectl-fqdn` binary in the project root. Copy it somewhere on your `PATH` to test it against a live cluster:

```bash
cp kubectl-fqdn ~/.local/bin/
kubectl fqdn svc -n default
```

## Running Checks

```bash
go test ./...       # unit tests
go vet ./...        # static analysis
```

The CI pipeline runs both of these on every push and pull request.

## Project Layout

```
cmd/
  main.go              # binary entry point — calls cli.Execute()
internal/
  cli/
    root.go            # cobra command definition, flags, argument validation
    run.go             # kubeconfig loading, cluster domain detection, dispatch
  resolver/
    resolver.go        # Result/Resolver/Lister interfaces and the type registry
    cluster.go         # reads cluster domain from CoreDNS ConfigMap
    service.go         # ServiceResolver
    pod.go             # PodResolver
    ingress.go         # IngressResolver
    node.go            # NodeResolver
  output/
    print.go           # tabwriter-based output formatting
```

## Adding a New Resource Type

1. Create `internal/resolver/<type>.go` and implement the two interfaces from `resolver.go`:

   ```go
   type Resolver interface {
       Resolve(ctx context.Context, client kubernetes.Interface, ns, name, domain string) []Result
   }

   type Lister interface {
       ListAll(ctx context.Context, client kubernetes.Interface, ns, domain string) []NamedResults
   }
   ```

2. Register the new resolver in `resolver.Registry` (in `resolver.go`), mapping every alias to the same instance.

3. Add the canonical type name to `AllTypes` in the same file so `kubectl fqdn all` picks it up.

4. Add examples to the README table.

## Commit Style

This project uses [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add ExternalName CNAME resolution
fix: handle pods with no assigned IP
docs: document --resolve flag behaviour
refactor: extract cluster domain detection
test: add service resolver edge cases
```

Keep the subject line under 72 characters. Add a body if the motivation is non-obvious.

## Opening a Pull Request

1. Fork the repository and create a feature branch from `main`:
   ```bash
   git checkout -b feat/my-feature
   ```
2. Make your changes, ensure `go test ./...` and `go vet ./...` pass.
3. Push your branch and open a PR against `main`.
4. Describe what the change does and why. Reference any related issues.
5. CI must be green before merging.

## Reporting Bugs / Requesting Features

Open a [GitHub Issue](https://github.com/imryanparsa/kfqdn/issues). For bugs, include:

- `kubectl fqdn` command that reproduces the problem
- Kubernetes version (`kubectl version`)
- Expected vs. actual output
