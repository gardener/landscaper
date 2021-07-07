# SPDX-FileCopyrightText: 2019 SAP SE or an SAP affiliate company and Gardener contributors.
#
# SPDX-License-Identifier: Apache-2.0

REPO_ROOT                                      := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
VERSION                                        := $(shell cat $(REPO_ROOT)/VERSION)
EFFECTIVE_VERSION                              := $(VERSION)-$(shell git rev-parse HEAD)

REGISTRY                                       := eu.gcr.io/gardener-project/landscaper
LANDSCAPER_CONTROLLER_IMAGE_REPOSITORY         := $(REGISTRY)/landscaper-controller
LANDSCAPER_WEBHOOKS_SERVER_IMAGE_REPOSITORY    := $(REGISTRY)/landscaper-webhooks-server
LANDSCAPER_AGENT_IMAGE_REPOSITORY              := $(REGISTRY)/landscaper-agent
CONTAINER_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY := $(REGISTRY)/container-deployer-controller
CONTAINER_DEPLOYER_INIT_IMAGE_REPOSITORY       := $(REGISTRY)/container-deployer-init
CONTAINER_DEPLOYER_WAIT_IMAGE_REPOSITORY       := $(REGISTRY)/container-deployer-wait
HELM_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY      := $(REGISTRY)/helm-deployer-controller
MANIFEST_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY  := $(REGISTRY)/manifest-deployer-controller
MOCK_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY      := $(REGISTRY)/mock-deployer-controller

DISABLE_CLEANUP := false

.PHONY: install-requirements
install-requirements:
	@go install -mod=vendor $(REPO_ROOT)/vendor/github.com/ahmetb/gen-crd-api-reference-docs
	@go install -mod=vendor $(REPO_ROOT)/vendor/github.com/golang/mock/mockgen
	@go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
	@$(REPO_ROOT)/hack/install-requirements.sh
	@chmod +x $(REPO_ROOT)/apis/vendor/k8s.io/code-generator/*

.PHONY: revendor
revendor:
	@$(REPO_ROOT)/hack/revendor.sh
	@cd $(REPO_ROOT)/apis && $(REPO_ROOT)/hack/revendor.sh
	@chmod +x $(REPO_ROOT)/apis/vendor/k8s.io/code-generator/*

.PHONY: format
format:
	@$(REPO_ROOT)/hack/format.sh $(REPO_ROOT)/apis/config $(REPO_ROOT)/apis/core $(REPO_ROOT)/apis/deployer $(REPO_ROOT)/pkg $(REPO_ROOT)/test $(REPO_ROOT)/cmd $(REPO_ROOT)/hack

.PHONY: check
check:
	@$(REPO_ROOT)/hack/check.sh --golangci-lint-config=./.golangci.yaml $(REPO_ROOT)/cmd/... $(REPO_ROOT)/pkg/... $(REPO_ROOT)/test/...
	@cd $(REPO_ROOT)/apis && $(REPO_ROOT)/hack/check.sh --golangci-lint-config=../.golangci.yaml $(REPO_ROOT)/apis/config/... $(REPO_ROOT)/apis/core/... $(REPO_ROOT)/apis/deployer/...

.PHONY: setup-testenv
setup-testenv:
	@$(REPO_ROOT)/hack/setup-testenv.sh

.PHONY: test
test: setup-testenv
	@$(REPO_ROOT)/hack/test.sh

.PHONY: integration-test
integration-test:
	@go test -timeout=1h -mod=vendor $(REPO_ROOT)/test/integration --v -ginkgo.v -ginkgo.progress \
		--kubeconfig $(KUBECONFIG) \
		--ls-version $(EFFECTIVE_VERSION) \
		--registry-config=$(REGISTRY_CONFIG) \
		--disable-cleanup=$(DISABLE_CLEANUP)


.PHONY: verify
verify: check

.PHONY: generate-code
generate-code:
	@cd $(REPO_ROOT)/apis && $(REPO_ROOT)/hack/generate.sh ./... && cd $(REPO_ROOT)
	@$(REPO_ROOT)/hack/generate.sh $(REPO_ROOT)/pkg... $(REPO_ROOT)/test... $(REPO_ROOT)/cmd...

.PHONY: generate
generate: generate-code format revendor

#################################################################
# Rules related to binary build, docker image build and release #
#################################################################

.PHONY: install
install:
	@EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) ./hack/install.sh

.PHONY: docker-images
docker-images:
	@echo "Building docker images for version $(EFFECTIVE_VERSION)"
	@docker build --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) -t $(LANDSCAPER_CONTROLLER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target landscaper-controller .
	@docker build --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) -t $(LANDSCAPER_WEBHOOKS_SERVER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target landscaper-webhooks-server .
	@docker build --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) -t $(LANDSCAPER_AGENT_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target landscaper-agent .
	@docker build --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) -t $(CONTAINER_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target container-deployer-controller .
	@docker build --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) -t $(CONTAINER_DEPLOYER_INIT_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target container-deployer-init .
	@docker build --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) -t $(CONTAINER_DEPLOYER_WAIT_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target container-deployer-wait .
	@docker build --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) -t $(HELM_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target helm-deployer-controller .
	@docker build --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) -t $(MANIFEST_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target manifest-deployer-controller .
	@docker build --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) -t $(MOCK_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target mock-deployer-controller .

.PHONY: docker-push
docker-push:
	@echo "Pushing docker images for version $(EFFECTIVE_VERSION) to registry $(REGISTRY)"
	@if ! docker images $(LANDSCAPER_CONTROLLER_IMAGE_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(EFFECTIVE_VERSION); then echo "$(LANDSCAPER_CONTROLLER_IMAGE_REPOSITORY) version $(EFFECTIVE_VERSION) is not yet built. Please run 'make docker-images'"; false; fi
	@if ! docker images $(LANDSCAPER_WEBHOOKS_SERVER_IMAGE_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(EFFECTIVE_VERSION); then echo "$(LANDSCAPER_WEBHOOKS_SERVER_IMAGE_REPOSITORY) version $(EFFECTIVE_VERSION) is not yet built. Please run 'make docker-images'"; false; fi
	@if ! docker images $(LANDSCAPER_AGENT_IMAGE_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(EFFECTIVE_VERSION); then echo "$(LANDSCAPER_AGENT_IMAGE_REPOSITORY) version $(EFFECTIVE_VERSION) is not yet built. Please run 'make docker-images'"; false; fi
	@if ! docker images $(CONTAINER_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(EFFECTIVE_VERSION); then echo "$(CONTAINER_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY) version $(EFFECTIVE_VERSION) is not yet built. Please run 'make docker-images'"; false; fi
	@if ! docker images $(CONTAINER_DEPLOYER_INIT_IMAGE_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(EFFECTIVE_VERSION); then echo "$(CONTAINER_DEPLOYER_INIT_IMAGE_REPOSITORY) version $(EFFECTIVE_VERSION) is not yet built. Please run 'make docker-images'"; false; fi
	@if ! docker images $(CONTAINER_DEPLOYER_WAIT_IMAGE_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(EFFECTIVE_VERSION); then echo "$(CONTAINER_DEPLOYER_WAIT_IMAGE_REPOSITORY) version $(EFFECTIVE_VERSION) is not yet built. Please run 'make docker-images'"; false; fi
	@if ! docker images $(HELM_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(EFFECTIVE_VERSION); then echo "$(HELM_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY) version $(EFFECTIVE_VERSION) is not yet built. Please run 'make docker-images'"; false; fi
	@if ! docker images $(MANIFEST_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(EFFECTIVE_VERSION); then echo "$(MANIFEST_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY) version $(EFFECTIVE_VERSION) is not yet built. Please run 'make docker-images'"; false; fi
	@if ! docker images $(MOCK_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(EFFECTIVE_VERSION); then echo "$(MOCK_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY) version $(EFFECTIVE_VERSION) is not yet built. Please run 'make docker-images'"; false; fi
	@docker push $(LANDSCAPER_CONTROLLER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION)
	@docker push $(LANDSCAPER_WEBHOOKS_SERVER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION)
	@docker push $(LANDSCAPER_AGENT_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION)
	@docker push $(CONTAINER_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION)
	@docker push $(CONTAINER_DEPLOYER_INIT_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION)
	@docker push $(CONTAINER_DEPLOYER_WAIT_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION)
	@docker push $(HELM_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION)
	@docker push $(MANIFEST_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION)
	@docker push $(MOCK_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION)

.PHONY: docker-all
docker-all: docker-images docker-push

.PHONY: cnudie
cnudie:
	@$(REPO_ROOT)/hack/generate-cd.sh

.PHONY: helm-charts
helm-charts:
	@$(REPO_ROOT)/.ci/publish-helm-charts

.PHONY: build-resources
build-resources: docker-all helm-charts cnudie

######################
# Tutorial resources #
######################

.PHONY: upload-tutorial-resources
upload-tutorial-resources:
	@./hack/upload-tutorial-resources.sh

######################
# Local development  #
######################

.PHONY: setup-local-registry
setup-local-registry:
	@go run ./hack/testcluster create --kubeconfig $(KUBECONFIG) -n default --id=local --enable-registry --enable-cluster=false --registry-auth=./tmp/local-docker.config
	@echo "For local development add '127.0.0.1 registry-local.default' to your '/etc/hosts' and run a port-forward to the registry pod 'kubectl port-forward registry-local 5000'"

.PHONY: remove-local-registry
remove-local-registry:
	@go run ./hack/testcluster delete --kubeconfig $(KUBECONFIG) -n default --id=local --enable-registry --enable-cluster=false

.PHONY: start-webhooks
start-webhooks:
	@go run ./cmd/landscaper-webhooks-server -v 3 --kubeconfig=$(KUBECONFIG)
