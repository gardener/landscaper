# SPDX-FileCopyrightText: 2019 SAP SE or an SAP affiliate company and Gardener contributors.
#
# SPDX-License-Identifier: Apache-2.0

SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

REPO_ROOT                                      := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
VERSION                                        := $(shell cat $(REPO_ROOT)/VERSION)
EFFECTIVE_VERSION                              := $(shell $(REPO_ROOT)/hack/get-version.sh)
BUILD_OS                                       := "linux"
BUILD_ARCH                                     := "amd64"

REGISTRY                                       := europe-docker.pkg.dev/sap-gcp-cp-k8s-stable-hub/landscaper
DOCKER_BUILDER_NAME := "ls-multiarch"

DISABLE_CLEANUP := false
ENVTEST_K8S_VERSION = 1.27

CODE_DIRS := $(REPO_ROOT)/cmd/... $(REPO_ROOT)/pkg/... $(REPO_ROOT)/test/... $(REPO_ROOT)/hack/testcluster/... $(REPO_ROOT)/apis/... $(REPO_ROOT)/controller-utils/...

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: revendor
revendor: ## Runs 'go mod tidy' for all go modules in this repo.
	@$(REPO_ROOT)/hack/revendor.sh

.PHONY: format
format: goimports ## Runs the formatter.
	@@FORMATTER=$(FORMATTER) $(REPO_ROOT)/hack/format.sh $(CODE_DIRS)

.PHONY: check
check: revendor golangci-lint jq goimports ## Runs linter, 'go vet', and checks if the formatter has been run.
	@test "$(SKIP_DOCS_INDEX_CHECK)" = "true" || \
		JQ=$(JQ) $(REPO_ROOT)/hack/verify-docs-index.sh
	@LINTER=$(LINTER) FORMATTER=$(FORMATTER) $(REPO_ROOT)/hack/check.sh --golangci-lint-config="$(REPO_ROOT)/.golangci.yaml" $(CODE_DIRS)

.PHONY: verify
verify: check ## Alias for 'make check'.

.PHONY: generate-code
generate-code: revendor code-gen controller-gen api-ref-gen mockgen ## Runs code generation (deepcopy/conversion/defaulter functions, API reference, openAPI definitions, CRDs, mock clients).
	@CODE_GEN_SCRIPT=$(CODE_GEN_SCRIPT) CONTROLLER_GEN=$(CONTROLLER_GEN) API_REF_GEN=$(API_REF_GEN) MOCKGEN=$(MOCKGEN) $(REPO_ROOT)/hack/generate-code.sh

.PHONY: generate-docs
generate-docs: jq ## Generates the documentation index.
	@JQ=$(JQ) $(REPO_ROOT)/hack/generate-docs-index.sh

.PHONY: generate # Runs code and docs generation and the formatter.
generate: generate-code format generate-docs

##@ Tests

.PHONY: test
test: envtest registry ## Runs unit tests.
	@KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" $(REPO_ROOT)/hack/test.sh

.PHONY: integration-test
integration-test: ## Runs integration tests.
	@$(REPO_ROOT)/hack/local-integration-test $(KUBECONFIG_PATH) $(EFFECTIVE_VERSION) $(USE_OCM_LIB)

.PHONY: integration-test-pure
integration-test-pure: ## Runs integration tests without installing the Landscaper first.
	@$(REPO_ROOT)/hack/local-integration-test-pure $(KUBECONFIG_PATH) $(EFFECTIVE_VERSION)

.PHONY: integration-test-with-cluster-creation
integration-test-with-cluster-creation: ## Runs integration tests and creates a cluster first.
	@$(REPO_ROOT)/hack/local-integration-test-with-cluster-creation $(KUBECONFIG_PATH) garden-laas $(EFFECTIVE_VERSION) 0 $(USE_OCM_LIB)

##@ Build

PLATFORMS ?= linux/arm64,linux/amd64

.PHONY: build
build: ## Build binaries for all os/arch combinations specified in PLATFORMS.
	@PLATFORMS=$(PLATFORMS) COMPONENT=landscaper-controller $(REPO_ROOT)/hack/build.sh
	@PLATFORMS=$(PLATFORMS) COMPONENT=landscaper-webhooks-server $(REPO_ROOT)/hack/build.sh
	@PLATFORMS=$(PLATFORMS) COMPONENT=container-deployer-controller COMPONENT_MAIN_PATH=container-deployer/container-deployer-controller $(REPO_ROOT)/hack/build.sh
	@PLATFORMS=$(PLATFORMS) COMPONENT=container-deployer-init COMPONENT_MAIN_PATH=container-deployer/container-deployer-init $(REPO_ROOT)/hack/build.sh
	@PLATFORMS=$(PLATFORMS) COMPONENT=container-deployer-wait COMPONENT_MAIN_PATH=container-deployer/container-deployer-wait $(REPO_ROOT)/hack/build.sh
	@PLATFORMS=$(PLATFORMS) COMPONENT=helm-deployer-controller $(REPO_ROOT)/hack/build.sh
	@PLATFORMS=$(PLATFORMS) COMPONENT=manifest-deployer-controller $(REPO_ROOT)/hack/build.sh
	@PLATFORMS=$(PLATFORMS) COMPONENT=mock-deployer-controller $(REPO_ROOT)/hack/build.sh
	@PLATFORMS=$(PLATFORMS) COMPONENT=target-sync-controller $(REPO_ROOT)/hack/build.sh
	
.PHONY: docker-images
docker-images: build ## Builds images for all controllers locally. The images are suffixed with -$OS-$ARCH
	@PLATFORMS=$(PLATFORMS) $(REPO_ROOT)/hack/docker-build-multi.sh

.PHONY: component
component: ocm ## Builds and pushes the Component Descriptor. Also pushes the images and combines them into multi-platform images. Requires the docker images to have been built before.
	@OCM=$(OCM) $(REPO_ROOT)/hack/generate-cd.sh $(REGISTRY)

.PHONY: build-resources ## Wrapper for 'make docker-images component'.
build-resources: docker-images component

##@ Tutorial Ressources

.PHONY: upload-tutorial-resources
upload-tutorial-resources: ## Uploads the resources that are referenced in the tutorial into a registry.
	@$(REPO_ROOT)/hack/upload-tutorial-resources.sh

##@ Local Development

.PHONY: install-testcluster-cmd
install-testcluster-cmd: ## Installs a k3s test cluster.
	@go install $(REPO_ROOT)/hack/testcluster

.PHONY: start-webhooks
start-webhooks: ## Runs the webhooks locally.
	@go run $(REPO_ROOT)/cmd/landscaper-webhooks-server -v 3 --kubeconfig=$(KUBECONFIG)

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(REPO_ROOT)/bin

## Tool Binaries
CODE_GEN_SCRIPT ?= $(LOCALBIN)/kube_codegen.sh
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
FORMATTER ?= $(LOCALBIN)/goimports
LINTER ?= $(LOCALBIN)/golangci-lint
OCM ?= $(LOCALBIN)/ocm
API_REF_GEN ?= $(LOCALBIN)/crd-ref-docs
MOCKGEN ?= $(LOCALBIN)/mockgen
JQ ?= $(LOCALBIN)/jq
REGISTRY_BINARY ?= $(LOCALBIN)/registry

## Tool Versions
CODE_GEN_VERSION ?= $(shell  $(REPO_ROOT)/hack/extract-module-version.sh k8s.io/code-generator)
# renovate: datasource=github-releases depName=kubernetes-sigs/controller-tools
CONTROLLER_TOOLS_VERSION ?= v0.15.0
# renovate: datasource=github-tags depName=golang/tools
FORMATTER_VERSION ?= v0.20.0
# renovate: datasource=github-releases depName=golangci/golangci-lint
LINTER_VERSION ?= v1.57.2
# renovate: datasource=github-releases depName=elastic/crd-ref-docs
API_REF_GEN_VERSION ?= v0.0.12
# renovate: datasource=github-releases depName=jqlang/jq
JQ_VERSION ?= v1.7.1
# renovate: datasource=github-releases depName=open-component-model/ocm
OCM_VERSION ?= v0.8.0
# renovate: datasource=github-releases depName=golang/mock
MOCKGEN_VERSION ?= v1.6.0
# renovate: datasource=github-releases depName=distribution/distribution
REGISTRY_VERSION ?= v3.0.0-alpha.1
# This cannot be handled properly e.g. with renovate, because the controller-runtime maintainers refuse to tag the
# setup-envtest module (https://github.com/kubernetes-sigs/controller-runtime/issues/2720)
SETUP_ENVTEST_VERSION ?= release-0.17

.PHONY: localbin
localbin: ## Creates the local bin folder, if it doesn't exist. Not meant to be called manually, used as requirement for the other tool commands.
	@test -d $(LOCALBIN) || mkdir -p $(LOCALBIN)

.PHONY: code-gen
code-gen: localbin ## Download the code-gen script locally.
	@test -s $(CODE_GEN_SCRIPT) && test -s $(LOCALBIN)/kube_codegen_version && cat $(LOCALBIN)/kube_codegen_version | grep -q $(CODE_GEN_VERSION) || \
	( echo "Downloading code generator script $(CODE_GEN_VERSION) ..."; \
	curl -sfL "https://raw.githubusercontent.com/kubernetes/code-generator/$(CODE_GEN_VERSION)/kube_codegen.sh" --output "$(CODE_GEN_SCRIPT)" && chmod +x "$(CODE_GEN_SCRIPT)" && \
	echo $(CODE_GEN_VERSION) > $(LOCALBIN)/kube_codegen_version )

.PHONY: controller-gen
controller-gen: localbin ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
	@test -s $(CONTROLLER_GEN) && $(CONTROLLER_GEN) --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	( echo "Installing controller-gen $(CONTROLLER_TOOLS_VERSION) ..."; \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION) )

.PHONY: goimports
goimports: localbin ## Download goimports locally if necessary. If wrong version is installed, it will be overwritten.
	@test -s $(FORMATTER) && test -s $(LOCALBIN)/goimports_version && cat $(LOCALBIN)/goimports_version | grep -q $(FORMATTER_VERSION) || \
	( echo "Installing goimports $(FORMATTER_VERSION) ..."; \
	GOBIN=$(LOCALBIN) go install golang.org/x/tools/cmd/goimports@$(FORMATTER_VERSION) && \
	echo $(FORMATTER_VERSION) > $(LOCALBIN)/goimports_version )

.PHONY: golangci-lint
golangci-lint: localbin ## Download golangci-lint locally if necessary. If wrong version is installed, it will be overwritten.
	@test -s $(LINTER) && $(LINTER) --version | grep -q $(subst v,,$(LINTER_VERSION)) || \
	( echo "Installing golangci-lint $(LINTER_VERSION) ..."; \
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(LOCALBIN) $(LINTER_VERSION) )

.PHONY: envtest
envtest: localbin ## Download envtest-setup locally.
	@echo "Installing setup-envtest ..."; \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@$(SETUP_ENVTEST_VERSION)

.PHONY: ocm
ocm: localbin ## Install OCM CLI if necessary. If wrong version is installed, it will be overwritten.
	@test -s $(OCM) && $(OCM) --version | grep -q $(subst v,,$(OCM_VERSION)) || \
	( echo "Installing OCM tooling $(OCM_VERSION) ..."; \
	curl -sSfL https://ocm.software/install.sh | OCM_VERSION=$(subst v,,$(OCM_VERSION)) bash -s $(LOCALBIN) )

.PHONY: api-ref-gen
api-ref-gen: localbin ## Download API reference generator locally if necessary. If wrong version is installed, it will be overwritten.
	@test -s $(API_REF_GEN) && test -s $(LOCALBIN)/crd-ref-docs_version && cat $(LOCALBIN)/crd-ref-docs_version | grep -q $(API_REF_GEN_VERSION) || \
	( echo "Installing API reference generator $(API_REF_GEN_VERSION) ..."; \
	GOBIN=$(LOCALBIN) go install github.com/elastic/crd-ref-docs@$(API_REF_GEN_VERSION) && \
	echo $(API_REF_GEN_VERSION) > $(LOCALBIN)/crd-ref-docs_version )

.PHONY: mockgen
mockgen: localbin ## Download mockgen locally if necessary. If wrong version is installed, it will be overwritten.
	@test -s $(MOCKGEN) && $(MOCKGEN) --version | grep -q $(MOCKGEN_VERSION) || \
	( echo "Installing mockgen ..."; \
	GOBIN=$(LOCALBIN) go install github.com/golang/mock/mockgen@$(MOCKGEN_VERSION) )

.PHONY: jq
jq: localbin ## Download jq locally if necessary. If wrong version is installed, it will be overwritten.
	@test -s $(JQ) && $(JQ) --version | grep -q $(subst v,,$(JQ_VERSION)) || \
	( echo "Installing jq $(JQ_VERSION) ..."; \
	JQ=$(JQ) LOCALBIN=$(LOCALBIN) $(REPO_ROOT)/hack/install-jq.sh $(subst v,,$(JQ_VERSION)) )

.PHONY: registry
registry: localbin ## Download registry locally if necessary. If wrong version is installed, it will be overwritten.
	@test -s $(REGISTRY_BINARY) && $(REGISTRY_BINARY) --version | grep -q $(REGISTRY_VERSION) || \
	( echo "Installing local registry ..."; \
	REGISTRY=$(REGISTRY_BINARY) LOCALBIN=$(LOCALBIN) $(REPO_ROOT)/hack/install-registry.sh $(REGISTRY_VERSION) )
