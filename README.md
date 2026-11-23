# odc-ironcore-machineset

A Kubernetes operator that runs a group of durable virtual machines on
[IronCore](https://github.com/ironcore-dev/ironcore) as one `MachineSet`.

> **Status: early development.** The API types (`MachineSet`, `MachineSetMember`) and their
> validation webhooks are in place. The controllers that create and restore the Machines are
> not implemented yet.

## Description

IronCore runs virtual machines as `Machine` objects, each with its own `Volume`s and
`NetworkInterface`s. A `MachineSet` (API group `compute.odc.telekom.com/v1alpha1`) describes a
number of identical Machines, called *members*, from one template.

Each member is recorded in a `MachineSetMember` object, which owns the member's Volumes and
NetworkInterfaces. When a member's Machine is lost or replaced, these resources and their IP
addresses stay with the member, so the replacement Machine gets the same disks and network
identity.

| Field | Purpose |
|---|---|
| `spec.replicas` | Number of members (default `1`) |
| `spec.template` | The Machine of every member: MachineClass, power state, volumes, network interfaces, ignition and more |
| `spec.volumeTemplates`, `spec.networkInterfaceTemplates` | Volumes and NetworkInterfaces created per member |
| `spec.resourceRetentionPolicy.whenScaledDown` | On scale-down, `Retain` keeps a member's resources, `Delete` removes them |

After creation, only `replicas`, `template.spec.power` and `resourceRetentionPolicy` can change.
Settings that belong to exactly one Machine, such as a hostname or a fixed IP address, are only
allowed while `replicas` is at most 1. See
[`config/samples/compute_v1alpha1_machineset.yaml`](config/samples/compute_v1alpha1_machineset.yaml)
for an example.

## Getting Started

### Prerequisites

To build:
- Go 1.26.11 or newer (see `go.mod`)
- Docker
- kubectl

To run, a Kubernetes cluster with:
- The [IronCore](https://github.com/ironcore-dev/ironcore) API (`compute`, `storage` and
  `networking` groups); the project builds against IronCore v0.9.0
- [cert-manager](https://cert-manager.io/), which issues the webhook and metrics certificates
- [Cilium](https://cilium.io/), for the network policies that allow traffic between the
  operator and the Kubernetes API server
- Optional: the [Prometheus Operator](https://prometheus-operator.dev/), to scrape metrics
  through the `ServiceMonitor`

The end-to-end tests run on [Kind](https://kind.sigs.k8s.io/) with Cilium 1.20.4,
cert-manager v1.21.3 and kube-prometheus-stack 92.14.1 (see the `Makefile`).

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/machineset-operator:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/machineset-operator:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create a MachineSet**
Apply the sample from `config/samples`. It expects a MachineClass, VolumeClass, Network and
prefix with the names used in the file; adjust them to your cluster first:

```sh
kubectl apply -k config/samples/
```

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following the options to release and provide this solution to the users.

### By providing a bundle with all YAML files (GitHub Releases)

Releases are cut from `main` by `.github/workflows/release.yml`. After a pull
request is merged, the version in the `VERSION` file is published as tag
`v<VERSION>`, unless that tag already exists. To release, bump `VERSION` in a
pull request.

No image is pushed to a registry. The first release, `v0.1.0`, is published with the first
merged pull request. Each release has these assets:

| Asset | Content |
|---|---|
| `install.yaml` | Everything needed to install this project, except the `ServiceMonitor` |
| `install-monitoring.yaml` | The `ServiceMonitor`; apply after the cert-manager certificates are `Ready` |
| `image-linux-<arch>.oci.tar` | The operator image (amd64, arm64) as OCI archive |
| `sbom-linux-<arch>.spdx.json`, `sbom-linux-<arch>.cyclonedx.json` | SBOMs of each image (Syft) |
| `SHA256SUMS` | Checksums of all assets |

The same artifacts can be built locally into `dist/`:

```sh
make release-artifacts IMG=machineset-operator:v0.1.0
```

The `ServiceMonitor` is split out because the Prometheus Operator rejects a
`ServiceMonitor` whose CA secret does not exist yet and does not re-evaluate it
later. Neither bundle contains the cluster dependencies (Cilium, cert-manager,
Prometheus Operator).

**Installing a release**

1. Download the assets and verify them:

```sh
TAG=v0.1.0

gh release download "$TAG" --repo telekom/odc-ironcore-machineset --dir "odc-ironcore-machineset-$TAG"
cd "odc-ironcore-machineset-$TAG"

shasum -a 256 -c SHA256SUMS
```

2. Mirror the image into your registry (e.g. inside the air gap):

```sh
REGISTRY=registry.example.internal/odc

skopeo copy \
    oci-archive:image-linux-amd64.oci.tar \
    "docker://$REGISTRY/machineset-operator:$TAG"
```

3. Point `install.yaml` at the mirrored image, then apply the bundles in order:

```sh
sed -i.bak "s|image: machineset-operator:|image: $REGISTRY/machineset-operator:|" install.yaml

kubectl apply -f install.yaml

kubectl wait certificates.cert-manager.io \
    --namespace odc-ironcore-machineset-system \
    --for=condition=Ready \
    --timeout=2m

kubectl apply -f install-monitoring.yaml
```

With GitOps, point two Flux `Kustomization`s (or Argo CD sync waves) at
`config/installer/operator` and `config/installer/monitoring`, with the second
depending on the first and `wait: true`.

## Contributing

Contributions are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md) and our
[Code of Conduct](CODE_OF_CONDUCT.md). To report a security vulnerability, follow
[SECURITY.md](SECURITY.md) and do not open a public issue.

Run `make help` to list all `make` targets. More background is in the
[Kubebuilder documentation](https://book.kubebuilder.io/introduction.html).

## License

Copyright 2026 Deutsche Telekom AG.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

