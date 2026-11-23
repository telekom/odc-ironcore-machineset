# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# YEAR defines the year value used for substituting the YEAR placeholder in the boilerplate header.
# It is pinned to the project's creation year so regenerating code does not churn existing headers.
YEAR ?= 2026

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	"$(CONTROLLER_GEN)" rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	"$(CONTROLLER_GEN)" object:headerFile="hack/boilerplate.go.txt",year=$(YEAR) paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet setup-envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell "$(ENVTEST)" use $(ENVTEST_K8S_VERSION) --bin-dir "$(LOCALBIN)" -p path)" go test $$(go list ./... | grep -v /e2e) -coverprofile cover.out

# TODO(user): To use a different vendor for e2e tests, modify the setup under 'tests/e2e'.
# The default setup assumes Kind is pre-installed and builds/loads the Manager Docker image locally.
# kubectl kuberc is disabled by default for test isolation; enable with:
# - KUBECTL_KUBERC=true
KIND_CLUSTER ?= odc-ironcore-machineset-test-e2e
KIND_CONFIG ?= hack/kind-config.yaml
KUBE_CONTEXT ?= kind-$(KIND_CLUSTER)
PROMETHEUS_NAMESPACE ?= monitoring
# Dedicated kubeconfig for the e2e cluster: ~/.kube/config and its current context are never
# touched, so the e2e tests cannot run against another cluster. Debug a kept cluster with:
#   KUBECONFIG=bin/kubeconfig-odc-ironcore-machineset-test-e2e kubectl get pods -A
E2E_KUBECONFIG ?= $(LOCALBIN)/kubeconfig-$(KIND_CLUSTER)

.PHONY: setup-test-e2e
setup-test-e2e: ## Set up a Kind cluster (cert-manager, Prometheus Operator) for e2e tests if it does not exist
	@command -v $(KIND) >/dev/null 2>&1 || { \
		echo "Kind is not installed. Please install Kind manually."; \
		exit 1; \
	}
	@mkdir -p "$(LOCALBIN)"
	@case "$$($(KIND) get clusters)" in \
		*"$(KIND_CLUSTER)"*) \
			echo "Kind cluster '$(KIND_CLUSTER)' already exists. Skipping creation." ;; \
		*) \
			echo "Creating Kind cluster '$(KIND_CLUSTER)'..."; \
			$(KIND) create cluster --name $(KIND_CLUSTER) --config $(KIND_CONFIG) \
				--kubeconfig "$(E2E_KUBECONFIG)" ;; \
	esac
	@# Also covers clusters created earlier with the default kubeconfig.
	$(KIND) get kubeconfig --name $(KIND_CLUSTER) > "$(E2E_KUBECONFIG)"
	KUBECONFIG="$(E2E_KUBECONFIG)" $(MAKE) install-cert-manager install-prometheus

.PHONY: install-cert-manager
install-cert-manager: ## Install cert-manager into the Kind cluster (KUBE_CONTEXT)
	@command -v $(HELM) >/dev/null 2>&1 || { \
		echo "Helm is not installed. Please install Helm manually."; \
		exit 1; \
	}
	$(HELM) upgrade --install cert-manager cert-manager \
		--repo https://charts.jetstack.io \
		--version $(CERT_MANAGER_VERSION) \
		--kube-context $(KUBE_CONTEXT) \
		--namespace cert-manager --create-namespace \
		--set crds.enabled=true \
		--wait --timeout 10m

.PHONY: install-prometheus
install-prometheus: ## Install a minimal Prometheus Operator stack into the Kind cluster (KUBE_CONTEXT)
	"$(KUBECTL)" --context $(KUBE_CONTEXT) create namespace $(PROMETHEUS_NAMESPACE) \
		--dry-run=client -o yaml | "$(KUBECTL)" --context $(KUBE_CONTEXT) apply -f -
	@# The optional metrics NetworkPolicy only admits scrapers from namespaces with this label.
	"$(KUBECTL)" --context $(KUBE_CONTEXT) label --overwrite namespace $(PROMETHEUS_NAMESPACE) metrics=enabled
	$(HELM) upgrade --install kube-prometheus-stack kube-prometheus-stack \
		--repo https://prometheus-community.github.io/helm-charts \
		--version $(KUBE_PROMETHEUS_STACK_VERSION) \
		--kube-context $(KUBE_CONTEXT) \
		--namespace $(PROMETHEUS_NAMESPACE) \
		--values hack/kube-prometheus-stack-values.yaml \
		--wait --timeout 10m

.PHONY: test-e2e
test-e2e: setup-test-e2e manifests generate fmt vet ## Run the e2e tests. Expected an isolated environment using Kind.
	KUBECONFIG="$(E2E_KUBECONFIG)" KIND=$(KIND) KIND_CLUSTER=$(KIND_CLUSTER) \
		go test -tags=e2e ./test/e2e/ -v -ginkgo.v
	$(MAKE) cleanup-test-e2e

.PHONY: cleanup-test-e2e
cleanup-test-e2e: ## Tear down the Kind cluster used for e2e tests
	@$(KIND) delete cluster --name $(KIND_CLUSTER) --kubeconfig "$(E2E_KUBECONFIG)"
	@rm -f "$(E2E_KUBECONFIG)"

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter
	"$(GOLANGCI_LINT)" run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
	"$(GOLANGCI_LINT)" run --fix

.PHONY: lint-config
lint-config: golangci-lint ## Verify golangci-lint linter configuration
	"$(GOLANGCI_LINT)" config verify

.PHONY: vulncheck
vulncheck: govulncheck ## Scan Go stdlib and module dependencies for known vulnerabilities.
	"$(GOVULNCHECK)" ./...

LICENSE_OWNER ?= Deutsche Telekom AG
LICENSE_FILES = $(shell find . -name '*.go' -not -path './bin/*')

.PHONY: add-license
add-license: addlicense ## Add Apache 2.0 license headers to Go files that lack one.
	"$(ADDLICENSE)" -c "$(LICENSE_OWNER)." -l apache -y "$(YEAR)" $(LICENSE_FILES)

.PHONY: check-license
check-license: addlicense ## Verify all Go files have a license header.
	"$(ADDLICENSE)" -check $(LICENSE_FILES)

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host (development logging, webhooks off).
	ENABLE_WEBHOOKS=false go run ./cmd/main.go --zap-devel=true

# If you wish to build the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	$(CONTAINER_TOOL) build -t ${IMG} .

.PHONY: sbom
sbom: syft ## Generate SPDX and CycloneDX SBOMs of ${IMG} into dist/ (run docker-build first).
	mkdir -p dist
	"$(SYFT)" scan "$(CONTAINER_TOOL):$(IMG)" \
		-o spdx-json=dist/sbom.spdx.json \
		-o cyclonedx-json=dist/sbom.cyclonedx.json

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	$(CONTAINER_TOOL) push ${IMG}

# PLATFORMS defines the target platforms for the manager image be built to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - be able to use docker buildx. More info: https://docs.docker.com/build/buildx/
# - have enabled BuildKit. More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image to your registry (i.e. if you do not set a valid value via IMG=<myregistry/image:<tag>> then the export will fail)
# To adequately provide solutions that are compatible with multiple platforms, you should consider using this option.
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
.PHONY: docker-buildx
docker-buildx: ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- $(CONTAINER_TOOL) buildx create --name odc-ironcore-machineset-builder
	$(CONTAINER_TOOL) buildx use odc-ironcore-machineset-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --tag ${IMG} -f Dockerfile.cross .
	- $(CONTAINER_TOOL) buildx rm odc-ironcore-machineset-builder
	rm Dockerfile.cross

.PHONY: build-installer
build-installer: manifests generate kustomize ## Generate dist/install.yaml and dist/install-monitoring.yaml (apply in that order), and the optional dist/install-network-policy.yaml.
	mkdir -p dist
	cd config/manager && "$(KUSTOMIZE)" edit set image controller=${IMG}
	@# Split like `make deploy`, see config/installer/operator/kustomization.yaml.
	"$(KUSTOMIZE)" build config/installer/operator > dist/install.yaml
	"$(KUSTOMIZE)" build config/installer/monitoring > dist/install-monitoring.yaml
	"$(KUSTOMIZE)" build config/installer/network-policy > dist/install-network-policy.yaml

# RELEASE_PLATFORMS lists the platforms for which `make release-artifacts` builds image archives and SBOMs.
RELEASE_PLATFORMS ?= linux/amd64 linux/arm64

.PHONY: release-artifacts
release-artifacts: build-installer syft ## Build installer bundles, per-platform OCI image archives, SBOMs and SHA256SUMS into dist/ (no push).
	@# Like docker-buildx: the builder stage runs natively and cross-compiles, so no emulation is needed.
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- $(CONTAINER_TOOL) buildx create --name odc-ironcore-machineset-release-builder
	@status=0; \
	image_name="$(IMG)"; image_name="$${image_name%:*}"; \
	image_tag="$(IMG)"; image_tag="$${image_tag##*:}"; \
	for platform in $(RELEASE_PLATFORMS); do \
		arch="$${platform##*/}"; \
		archive="dist/image-linux-$${arch}.oci.tar"; \
		echo "Building $(IMG) for $${platform} into $${archive}"; \
		$(CONTAINER_TOOL) buildx build --builder odc-ironcore-machineset-release-builder \
			--platform "$${platform}" -f Dockerfile.cross \
			--output "type=oci,dest=$${archive},name=$(IMG)" . \
		&& "$(SYFT)" scan "oci-archive:$${archive}" \
			--source-name "$${image_name}" --source-version "$${image_tag}" \
			-o "spdx-json=dist/sbom-linux-$${arch}.spdx.json" \
			-o "cyclonedx-json=dist/sbom-linux-$${arch}.cyclonedx.json" \
		|| { status=1; break; }; \
	done; \
	$(CONTAINER_TOOL) buildx rm odc-ironcore-machineset-release-builder; \
	rm -f Dockerfile.cross; \
	exit $${status}
	cd dist && shasum -a 256 *.yaml *.oci.tar *.json > SHA256SUMS

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	@out="$$( "$(KUSTOMIZE)" build config/crd 2>/dev/null || true )"; \
	if [ -n "$$out" ]; then echo "$$out" | "$(KUBECTL)" apply -f -; else echo "No CRDs to install; skipping."; fi

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	@out="$$( "$(KUSTOMIZE)" build config/crd 2>/dev/null || true )"; \
	if [ -n "$$out" ]; then echo "$$out" | "$(KUBECTL)" delete --ignore-not-found=$(ignore-not-found) -f -; else echo "No CRDs to delete; skipping."; fi

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && "$(KUSTOMIZE)" edit set image controller=${IMG}
	@# The ServiceMonitor is applied only after cert-manager issued its CA secret,
	@# see config/installer/operator/kustomization.yaml.
	"$(KUSTOMIZE)" build config/installer/operator | "$(KUBECTL)" apply -f -
	"$(KUBECTL)" wait certificates.cert-manager.io --all-namespaces \
		-l app.kubernetes.io/name=odc-ironcore-machineset --for=condition=Ready --timeout=2m
	"$(KUSTOMIZE)" build config/installer/monitoring | "$(KUBECTL)" apply -f -

.PHONY: deploy-network-policy
deploy-network-policy: kustomize ## Apply the optional NetworkPolicies (config/network-policy) to the K8s cluster specified in ~/.kube/config.
	"$(KUSTOMIZE)" build config/installer/network-policy | "$(KUBECTL)" apply -f -

.PHONY: undeploy
undeploy: kustomize ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	"$(KUSTOMIZE)" build config/default | "$(KUBECTL)" delete --ignore-not-found=$(ignore-not-found) -f -

##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p "$(LOCALBIN)"

## Tool Binaries
KUBECTL ?= kubectl
KIND ?= kind
HELM ?= helm
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint
GOVULNCHECK ?= $(LOCALBIN)/govulncheck
ADDLICENSE ?= $(LOCALBIN)/addlicense
SYFT ?= $(LOCALBIN)/syft

## Tool Versions
KUSTOMIZE_VERSION ?= v5.8.1
CONTROLLER_TOOLS_VERSION ?= v0.22.0

#ENVTEST_VERSION is the controller-runtime version to use for setup-envtest, derived from go.mod
ENVTEST_VERSION ?= $(shell v='$(call gomodver,sigs.k8s.io/controller-runtime)'; \
  [ -n "$$v" ] || { echo "Set ENVTEST_VERSION manually (controller-runtime replace has no tag)" >&2; exit 1; }; \
  printf '%s\n' "$$v")

#ENVTEST_K8S_VERSION is the version of Kubernetes to use for setting up ENVTEST binaries (i.e. 1.31)
ENVTEST_K8S_VERSION ?= $(shell v='$(call gomodver,k8s.io/api)'; \
  [ -n "$$v" ] || { echo "Set ENVTEST_K8S_VERSION manually (k8s.io/api replace has no tag)" >&2; exit 1; }; \
  printf '%s\n' "$$v" | sed -E 's/^v?[0-9]+\.([0-9]+).*/1.\1/')

GOLANGCI_LINT_VERSION ?= v2.15.0
GOVULNCHECK_VERSION ?= v1.9.0
ADDLICENSE_VERSION ?= v1.2.0
SYFT_VERSION ?= v1.55.0

## Helm chart versions for the Kind cluster (setup-test-e2e)
CERT_MANAGER_VERSION ?= v1.21.3
KUBE_PROMETHEUS_STACK_VERSION ?= 92.14.1

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: setup-envtest
setup-envtest: envtest ## Download the binaries required for ENVTEST in the local bin directory.
	@echo "Setting up envtest binaries for Kubernetes version $(ENVTEST_K8S_VERSION)..."
	@"$(ENVTEST)" use $(ENVTEST_K8S_VERSION) --bin-dir "$(LOCALBIN)" -p path || { \
		echo "Error: Failed to set up envtest binaries for version $(ENVTEST_K8S_VERSION)."; \
		exit 1; \
	}

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))
	@test -f .custom-gcl.yml && { \
		echo "Building custom golangci-lint with plugins..." && \
		$(GOLANGCI_LINT) custom --destination $(LOCALBIN) --name golangci-lint-custom && \
		mv -f $(LOCALBIN)/golangci-lint-custom $(GOLANGCI_LINT); \
	} || true

.PHONY: govulncheck
govulncheck: $(GOVULNCHECK) ## Download govulncheck locally if necessary.
$(GOVULNCHECK): $(LOCALBIN)
	$(call go-install-tool,$(GOVULNCHECK),golang.org/x/vuln/cmd/govulncheck,$(GOVULNCHECK_VERSION))

.PHONY: addlicense
addlicense: $(ADDLICENSE) ## Download addlicense locally if necessary.
$(ADDLICENSE): $(LOCALBIN)
	$(call go-install-tool,$(ADDLICENSE),github.com/google/addlicense,$(ADDLICENSE_VERSION))

.PHONY: syft
syft: $(SYFT) ## Download syft locally if necessary.
$(SYFT): $(LOCALBIN)
	@# Stamp the version like upstream releases do, so the generated SBOMs name the exact tool version.
	$(call go-install-tool,$(SYFT),github.com/anchore/syft/cmd/syft,$(SYFT_VERSION),-ldflags='-X main.version=$(SYFT_VERSION:v%=%)')

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] && [ "$$(readlink -- "$(1)" 2>/dev/null)" = "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f "$(1)" ;\
GOBIN="$(LOCALBIN)" go install $(4) $${package} ;\
mv "$(LOCALBIN)/$$(basename "$(1)")" "$(1)-$(3)" ;\
} ;\
ln -sf "$$(realpath "$(1)-$(3)")" "$(1)"
endef

define gomodver
$(shell go list -m -f '{{if .Replace}}{{.Replace.Version}}{{else}}{{.Version}}{{end}}' $(1) 2>/dev/null)
endef
