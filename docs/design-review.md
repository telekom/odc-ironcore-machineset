---
marp: true
theme: default
paginate: true
size: 16:9
header: "odc-ironcore-machineset · Design Review"
footer: "telekom/odc-ironcore-machineset · 2026-10"
style: |
  section {
    font-family: "Inter", "Helvetica Neue", Arial, sans-serif;
    font-size: 24px;
    padding: 50px 60px;
    color: #1f2937;
  }
  h1 { color: #0f3d6e; font-size: 40px; margin-bottom: 10px; }
  h2 { color: #1d6fb8; font-size: 28px; }
  table { font-size: 19px; width: 100%; }
  th { background: #0f3d6e; color: #fff; }
  code { font-size: 0.9em; }
  pre { font-size: 15px; line-height: 1.25; background: #f3f6fa; border-left: 4px solid #1d6fb8; }
  section.lead { background: linear-gradient(135deg, #0f3d6e 0%, #1d6fb8 100%); color: #fff; }
  section.lead h1, section.lead h2 { color: #fff; }
  section.lead header, section.lead footer { color: #dbe7f5; }
  .cols { display: grid; grid-template-columns: 1fr 1fr; gap: 30px; }
  .tag { display: inline-block; padding: 2px 10px; border-radius: 10px; font-size: 16px; color: #fff; }
  .ok { background: #15803d; } .warn { background: #b45309; } .q { background: #6b7280; }
  .small { font-size: 18px; color: #4b5563; }
  section.dense { padding-top: 40px; padding-bottom: 40px; }
  section.dense h1 { font-size: 34px; margin-bottom: 6px; }
  section.dense table { font-size: 15px; }
  section.dense th, section.dense td { padding: 4px 10px; }
  section.toc { font-size: 21px; padding-top: 40px; padding-bottom: 40px; }
  section.toc h1 { margin-bottom: 0; }
  section.toc p { margin: 12px 0 4px; }
  section.toc ol { margin: 0; }
---

<!-- _class: lead -->
<!-- _paginate: false -->

# odc-ironcore-machineset

## Design Review: Platform, Security & Delivery Setup

Kubernetes operator for **Open Defence Cloud (ODC)**
Sovereign · air-gapped · hardened — IronCore as optional target

---

<!-- _class: toc -->

# Table of Contents

<div class="cols">
<div>

**Overview**

3. [Context & Design Principles](#context--design-principles)
4. [Runtime Architecture](#runtime-architecture)

**Security**

5. [Network Security: Default-Deny Firewall](#network-security-default-deny-firewall)
6. [Metrics and Webhook TLS: Verified, Not Skipped](#metrics-and-webhook-tls-verified-not-skipped)
7. [Install Ordering: The ServiceMonitor Race](#install-ordering-the-servicemonitor-race)

**Workflow**

8. [Development & Test Workflow](#development--test-workflow)
9. [Make Targets (1/2): Development & Testing](#make-targets-12-development--testing)
10. [Make Targets (2/2): Build, Release & Deployment](#make-targets-22-build-release--deployment)
11. [CI Workflows & Branch Rules](#ci-workflows--branch-rules)

</div>
<div>

**Supply Chain**

12. [Supply Chain & Quality Gates](#supply-chain--quality-gates)
13. [Image Scanning & SBOM: Trivy and Syft](#image-scanning--sbom-trivy-and-syft)

**Outcome**

14. [Key Decisions](#key-decisions)
15. [Open Issues](#open-issues)
16. [Questions & Discussion](#questions--discussion)

</div>
</div>

---

# Context & Design Principles

<div class="cols">
<div>

## Targets

- **Primary:** Open Defence Cloud (ODC)
  sovereign, air-gapped, hardened
- **Secondary:** IronCore (`ironcore-dev`) compatibility, decoupled
- Scaffolded with **Kubebuilder v4**
  domain `odc.telekom.com`

</div>
<div>

## Principles

- **Minimal tech debt** — stay on upstream conventions
- **Minimal runtime deps** — `go.mod` limited to controller-runtime / client-go / apimachinery
- **Secure by default** — deny-all network, verified TLS, non-root
- **Reproducible** — every tool, chart, image and CI binary pinned

</div>
</div>

---

# Runtime Architecture

```text
┌──────────────────────────────── Kubernetes cluster (CNI: Cilium) ────────────────────────────────┐
│                                                                                                  │
│  ns: odc-ironcore-machineset-system              ns: monitoring  (label metrics=enabled)         │
│  ┌──────────────────────────────────────┐        ┌────────────────────────────────────┐          │
│  │ Deployment controller-manager        │ HTTPS  │ Prometheus  (kube-prometheus-stack)│          │
│  │  · distroless:nonroot, read-only FS  │◄───────│  scrapes via ServiceMonitor        │          │
│  │  · TLS 1.3 metrics :8443 + authn/z   │ :8443  │  verifies cert with CA (no -k)     │          │
│  │  · validating webhook :9443 (TLS)    │        └────────────────────────────────────┘          │
│  └───────┬──────────────────────▲───────┘                                                        │
│          │ mounts               │                                                                │
│  ┌───────▼──────────────┐  ┌────┴─────────────────────────────┐                                  │
│  │ Secrets              │  │ NetworkPolicies                  │                                  │
│  │ metrics-server-cert  │  │  default-deny-all (in + out)     │      ns: kube-system             │
│  │ webhook-server-cert  │  │  allow-metrics-traffic  ◄────────┼──── from metrics=enabled ns      │
│  │ (tls.crt/key, ca.crt)│  │  allow-dns-egress ───────────────┼───► kube-dns :53                 │
│  └───────▲──────────────┘  │  allow-apiserver-egress (Cilium) ┼───► kube-apiserver :443/6443     │
│          │ issues          │  allow-webhook-ingress (Cilium) ◄┼──── kube-apiserver → :9443       │
│  ┌───────┴──────────────┐  └──────────────────────────────────┘                                  │
│  │ cert-manager         │                                                                        │
│  │ self-signed → CA →   │                                                                        │
│  │ serving certificate  │                                                                        │
│  └──────────────────────┘                                                                        │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

# Network Security: Default-Deny Firewall

| Policy                   | Kind                    | Allows                                         |
| ------------------------ | ----------------------- | ---------------------------------------------- |
| `default-deny-all`       | NetworkPolicy           | nothing — blocks all ingress & egress          |
| `allow-dns-egress`       | NetworkPolicy           | operator → `kube-dns` on 53 UDP/TCP            |
| `allow-metrics-traffic`  | NetworkPolicy           | namespaces labelled `metrics=enabled` → :8443  |
| `allow-apiserver-egress` | **CiliumNetworkPolicy** | operator → `kube-apiserver` entity on 443/6443 |
| `allow-webhook-ingress`  | **CiliumNetworkPolicy** | `kube-apiserver` entity → webhook on :9443     |

**Why two Cilium-specific policies?**
The API server runs on the host network and its Service IP is translated _before_ policy evaluation → not selectable portably with standard `NetworkPolicy`. This applies in both directions: the operator calling the API server, and the API server calling our validating webhook.
Alternative (`policyCIDRMatchMode=nodes` + per-environment CIDRs) **rejected**: environment-specific IPs, fragile.

<span class="small">Verified in e2e: allowed namespace succeeds, `default` namespace is blocked.</span>

---

# Metrics and Webhook TLS: Verified, Not Skipped

<div class="cols">
<div>

## Certificate chain (`config/certmanager`)

```text
Issuer  selfsigned-issuer
   │
   ▼
Certificate  metrics-ca  (isCA, ECDSA P-256)
   │  → Secret metrics-ca
   ▼
Issuer  metrics-ca-issuer
   │
   ├──► Certificate  metrics-certs
   │      → Secret metrics-server-cert
   │        (tls.crt, tls.key, ca.crt)
   │
   └──► Certificate  serving-cert  (webhook)
          → Secret webhook-server-cert
            ca.crt injected into the
            ValidatingWebhookConfiguration
```

</div>
<div>

## Result

- `ServiceMonitor` uses `insecureSkipVerify: false`, `serverName` = service DNS, CA from secret
- No external CA needed → works **air-gapped**
- In ODC: swap the self-signed root for the **environment's internal CA**
- Metrics server loads certs from the mounted secret (`--metrics-cert-path`); controller-runtime's cert watcher picks up rotations without restart
- The **webhook** certificate comes from the same CA. cert-manager's CA injector writes the CA into the `ValidatingWebhookConfiguration`, so the API server can verify the webhook over TLS (`--webhook-cert-path`, also reloaded on rotation)

</div>
</div>

---

# Install Ordering: The ServiceMonitor Race

**Finding:** Prometheus Operator _rejects_ a `ServiceMonitor` whose CA secret does not yet exist — and **never re-evaluates** it when the secret appears (only on spec change or operator restart).

**Decision:** two-phase installation, expressed declaratively as Kustomize overlays

```text
config/installer/operator     everything except the ServiceMonitor   ──► apply first
            │
            ▼   wait: Certificates Ready
config/installer/monitoring   only the ServiceMonitor                ──► apply second
```

| Consumer               | How ordering is enforced                                                 |
| ---------------------- | ------------------------------------------------------------------------ |
| `make deploy`          | apply → `kubectl wait` certificates → apply                              |
| `make build-installer` | `dist/install.yaml` + `dist/install-monitoring.yaml`                     |
| GitOps                 | Flux `dependsOn` + `wait: true` / Argo CD sync waves on the two overlays |

---

# Development & Test Workflow

<div class="cols">
<div>

## `make setup-test-e2e`

1. kind cluster, default CNI disabled
2. `install-cilium` (Helm, pinned)
3. `install-cert-manager` (Helm, pinned)
4. `install-prometheus` — minimal kube-prometheus-stack

**Isolation:** dedicated kubeconfig
`bin/kubeconfig-<cluster>` → `~/.kube/config` and other clusters are **never touched**

</div>
<div>

## `make test-e2e` — 4 specs

- ✅ controller pod runs (restricted PSS)
- ✅ metrics reachable from allowed namespace
- ✅ metrics **blocked** from other namespaces
- ✅ Prometheus scrapes with **verified TLS** (`up == 1`)

Same Make targets for laptop and CI → no drift between dev and pipeline.

</div>
</div>

---

<!-- _class: dense -->

# Make Targets (1/2): Development & Testing

| Target                                                           | Summary                                                                                 |
| ---------------------------------------------------------------- | --------------------------------------------------------------------------------------- |
| `help`                                                           | List all targets with their descriptions                                                |
| `manifests`                                                      | Generate CRDs, RBAC (ClusterRole) and webhook configs from Go markers                   |
| `generate`                                                       | Generate `DeepCopy` methods (`YEAR`-pinned license header)                              |
| `fmt` / `vet`                                                    | `go fmt` and `go vet`                                                                   |
| `test`                                                           | Unit and envtest integration tests (real kube-apiserver + etcd)                         |
| `lint` / `lint-fix` / `lint-config`                              | golangci-lint v2 with `gosec`; auto-fix; validate config                                |
| `vulncheck`                                                      | govulncheck on Go stdlib and module dependencies                                        |
| `add-license` / `check-license`                                  | Add or verify Apache 2.0 headers                                                        |
| `setup-test-e2e`                                                 | Kind cluster (default CNI off) + Cilium, cert-manager, Prometheus; dedicated kubeconfig |
| `install-cilium` / `install-cert-manager` / `install-prometheus` | Pinned Helm installs into the Kind cluster                                              |
| `test-e2e`                                                       | e2e suite: deploy, metrics, NetworkPolicy, Prometheus scrape with verified TLS          |
| `cleanup-test-e2e`                                               | Delete the Kind cluster and its kubeconfig                                              |

---

<!-- _class: dense -->

# Make Targets (2/2): Build, Release & Deployment

| Target                                                                                                          | Summary                                                                                                 |
| --------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| `build` / `run`                                                                                                 | Build the manager binary / run it locally (development logging, webhooks off)                           |
| `docker-build` / `docker-push`                                                                                  | Build / push the distroless image (`IMG`)                                                               |
| `docker-buildx`                                                                                                 | Multi-platform build and push (`PLATFORMS`)                                                             |
| `sbom`                                                                                                          | SPDX + CycloneDX SBOMs of the local image into `dist/`                                                  |
| `build-installer`                                                                                               | `dist/install.yaml` + `dist/install-monitoring.yaml` (two-phase)                                        |
| `release-artifacts`                                                                                             | Installer bundles, per-arch OCI image archives, SBOMs, `SHA256SUMS`; no push (used by release workflow) |
| `install` / `uninstall`                                                                                         | Apply / delete CRDs                                                                                     |
| `deploy`                                                                                                        | Two-phase deploy: operator → wait for certificates → ServiceMonitor                                     |
| `undeploy`                                                                                                      | Remove all deployed resources                                                                           |
| `kustomize`, `controller-gen`, `envtest`, `setup-envtest`, `golangci-lint`, `govulncheck`, `addlicense`, `syft` | Install the version-pinned tool into `bin/` on first use                                                |

---

<!-- _class: dense -->

# CI Workflows & Branch Rules

| Workflow    | Trigger                                                  | Check name                  | Purpose                                                    |
| ----------- | -------------------------------------------------------- | --------------------------- | ---------------------------------------------------------- |
| `lint`      | `push` to feature branches (not `main`)                  | `Lint`                      | `golangci-lint` incl. `gosec`                              |
| `license`   | PR → `main`: `opened`, `synchronize`, `reopened`         | `License headers`           | `addlicense` header check                                  |
| `test`      | PR → `main`: `opened`, `synchronize`, `reopened`         | `Unit tests`                | `go mod tidy -diff`, `make test` (envtest)                 |
| `test-e2e`  | PR → `main`: `opened`, `synchronize`, `reopened`         | `E2E tests`                 | Kind + Cilium + cert-manager + Prometheus, 4 specs         |
| `vulncheck` | PR → `main`: `opened`, `synchronize`, `reopened`         | `Vulnerability scan`        | `govulncheck` (stdlib + modules)                           |
| `release`   | PR → `main`: `closed` **and merged**                     | `Publish release artifacts` | Tag `v<VERSION>`, attach images, SBOMs, `SHA256SUMS`       |

- Checks run **only on open PRs**: no `push` triggers, so closing or merging a PR does not re-run them
- Unique job names → each check can be selected as a **required status check**

**Branch rule on `main`** (GitHub ruleset, not a workflow):

- **Require status checks to pass**: `Unit tests`, `E2E tests`, `Vulnerability scan`, `License headers`, `Lint`
- **Require branches to be up to date before merging** → PR must contain the latest `main`
- **Require linear history** → no merge commits, so "up to date" means **rebased on `main`**

---

# Supply Chain & Quality Gates

| Area             | Measure                                                                                          |
| ---------------- | ------------------------------------------------------------------------------------------------ |
| Go code          | `golangci-lint` v2 **with `gosec`**, `go vet`, `go mod tidy -diff`                               |
| Vulnerabilities  | `govulncheck` (stdlib + modules) in CI                                                           |
| Licensing        | `addlicense` check in CI, Apache 2.0 headers, `YEAR` pinned (no churn)                           |
| Build tools      | controller-gen, kustomize, envtest, golangci-lint, addlicense, syft pinned in `Makefile` → `bin/`      |
| Container images | `golang:1.26.11` and `distroless/static:nonroot` pinned **by digest**; e2e curl pinned by digest |
| Helm charts      | Cilium 1.20.4 · cert-manager v1.21.3 · kube-prometheus-stack 92.14.1                             |
| CI binaries      | kind v0.33.0 & Helm v4.3.1 verified with **pinned SHA-256**                                      |
| Runtime          | TLS 1.3 minimum, non-root, read-only root FS, drop ALL caps, production logging default          |
| Releases | Per-arch OCI image archives + SPDX/CycloneDX SBOMs (Syft) + `SHA256SUMS` on GitHub Releases |

CI workflows: `lint` · `test` · `test-e2e` · `vulncheck` · `license` · `release` (all on `ubuntu-26.04`)

---

# Image Scanning & SBOM: Trivy and Syft

| Tool      | Purpose                                                                                                                                     | Gap it would close                                                                                                                                                |
| --------- | ------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Trivy** | Scans the **built container image** for known CVEs and misconfigurations                                                                    | `govulncheck` covers Go code only; Trivy also checks the base image (`distroless/static`: CA certificates, tzdata) and can lint Kubernetes manifests / Dockerfile |
| **Syft**  | Generates an **SBOM**(Software Bill of Materials): a machine-readable inventory of every component in the image, with versions and licenses | Answers _"what exactly do we ship?"_ for audits and later CVE lookups                                                                                             |

Outcome: **Syft adopted** (`make sbom`), **Trivy not planned** → see _Key Decisions_.

---

<!-- _class: dense -->

# Key Decisions

| Decision                                           | Rationale                                                                                                                                          | Alternative considered                     |
| -------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------ |
| **Cilium** as CNI                                  | Supported by NeoNephos Foundation via Gardener                                                                                                     | kindnet (no policy enforcement)            |
| **Standard NetworkPolicy** + 2 CiliumNetworkPolicy | Portable; Cilium only where the standard API cannot express it (traffic to and from the API server)                                                 | CIDR-based rules per environment           |
| **Validation in Go** (validating webhook), no CEL  | No cost budget, so no artificial size limits; the same Go functions move into `ironcore-apiserver` if MachineSet joins IronCore (#20). CRD markers keep only shape and defaults | CEL rules in the CRD (bounded lists, cost budget, rewrite needed for IronCore) |
| **Prometheus via Helm** (minimal stack)            | Not our focus — we don't maintain a Prometheus server                                                                                              | Own manifests / Prometheus operator bundle |
| **cert-manager** self-signed CA                    | Real TLS verification without external CA; swappable in ODC                                                                                        | `insecureSkipVerify: true`                 |
| **Two-phase install** overlays                     | Avoids ServiceMonitor race in every delivery path                                                                                                  | Restart Prometheus Operator after deploy   |
| **Pin everything** (tags + digests + SHA-256)      | Reproducible, tamper-evident, air-gap friendly                                                                                                     | Floating `latest` tags                     |
| **Dedicated e2e kubeconfig**                       | Tests can never hit a wrong / production cluster                                                                                                   | Rely on current context                    |
| **Syft** SBOM per release, **no Trivy**            | We ship one static Go binary on distroless, already covered by `govulncheck`; the SBOM documents each release for ODC audits and later CVE lookups | Trivy image scan (adds little value here)  |
| **Releases from `main`** via `VERSION` file        | A merged PR publishes `v<VERSION>` if the tag doesn't exist yet; versioning is reviewed like code                                                  | Tag pushes / automatic bump per merge      |
| **No registry push**: release assets only          | Image as per-arch OCI archive + SBOMs + `SHA256SUMS` on GitHub Releases; verifiable and mirrorable into air-gapped registries                      | Push to ghcr.io                            |

---

<!-- _class: dense -->

# Open Issues

| #   | Item                                                                                                                  | Status                                |
| --- | --------------------------------------------------------------------------------------------------------------------- | ------------------------------------- |
| 1   | Run e2e workflow on GitHub — never executed there; `ubuntu-26.04` is a preview image                                  | <span class="tag warn">open</span>    |
| 2   | Nightly `govulncheck` run (catch new CVEs without code changes)                                                       | <span class="tag warn">open</span>    |
| 3   | Dependency update automation: `dependabot.yml` / Renovate for Makefile & workflow pins                                | <span class="tag q">to discuss</span> |
| 4   | Cluster prerequisites for `make deploy`: Cilium, cert-manager, Prometheus Operator CRDs — confirm availability in ODC | <span class="tag q">to discuss</span> |

<span class="small">Validation needs the webhook running: with `failurePolicy: Fail`, MachineSets can't be created or changed while the operator is down (running VMs are not affected).</span>

---

<!-- _class: lead -->
<!-- _paginate: false -->

# Questions & Discussion

Open for discussion and testing:

- Cilium-specific API-server egress policy
