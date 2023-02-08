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

DOCKER_BUILDER_NAME := "ls-multiarch"

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
	@cd $(REPO_ROOT)/controller-utils && $(REPO_ROOT)/hack/revendor.sh
	@chmod +x $(REPO_ROOT)/apis/vendor/k8s.io/code-generator/*

.PHONY: format
format:
	@$(REPO_ROOT)/hack/format.sh $(REPO_ROOT)/apis/config $(REPO_ROOT)/apis/core $(REPO_ROOT)/apis/deployer $(REPO_ROOT)/apis/errors $(REPO_ROOT)/apis/mediatype $(REPO_ROOT)/apis/openapi $(REPO_ROOT)/apis/schema $(REPO_ROOT)/pkg $(REPO_ROOT)/test $(REPO_ROOT)/cmd $(REPO_ROOT)/hack $(REPO_ROOT)/controller-utils/pkg

.PHONY: check
check: format
	@$(REPO_ROOT)/hack/verify-docs-index.sh
	@$(REPO_ROOT)/hack/check.sh --golangci-lint-config=./.golangci.yaml $(REPO_ROOT)/hack/testcluster/...
	@$(REPO_ROOT)/hack/check.sh --golangci-lint-config=./.golangci.yaml $(REPO_ROOT)/cmd/... $(REPO_ROOT)/pkg/... $(REPO_ROOT)/test/...
	@cd $(REPO_ROOT)/apis && $(REPO_ROOT)/hack/check.sh --golangci-lint-config=../.golangci.yaml $(REPO_ROOT)/apis/config/... $(REPO_ROOT)/apis/core/... $(REPO_ROOT)/apis/deployer/... $(REPO_ROOT)/apis/errors/... $(REPO_ROOT)/apis/mediatype/... $(REPO_ROOT)/apis/openapi/... $(REPO_ROOT)/apis/schema/...
	@cd $(REPO_ROOT)/controller-utils && $(REPO_ROOT)/hack/check.sh --golangci-lint-config=../.golangci.yaml $(REPO_ROOT)/controller-utils/pkg/...

.PHONY: setup-testenv
setup-testenv:
	@$(REPO_ROOT)/hack/setup-testenv.sh

.PHONY: test
test: setup-testenv
	@$(REPO_ROOT)/hack/test.sh


.PHONY: integration-test
integration-test:
	@$(REPO_ROOT)/.ci/local-integration-test $(KUBECONFIG_PATH) $(EFFECTIVE_VERSION)

.PHONY: integration-test-pure
integration-test-pure:
	@$(REPO_ROOT)/.ci/local-integration-test-pure $(KUBECONFIG_PATH) $(EFFECTIVE_VERSION)

.PHONY: integration-test-with-cluster-creation
integration-test-with-cluster-creation:
	@$(REPO_ROOT)/.ci/local-integration-test-with-cluster-creation $(KUBECONFIG_PATH) garden-laas $(EFFECTIVE_VERSION) 0

.PHONY: verify
verify: check

.PHONY: generate-code
generate-code:
	@cd $(REPO_ROOT)/apis && $(REPO_ROOT)/hack/generate.sh ./... && cd $(REPO_ROOT)
	@$(REPO_ROOT)/hack/generate.sh $(REPO_ROOT)/pkg... $(REPO_ROOT)/test... $(REPO_ROOT)/cmd...

.PHONY: generate-docs
generate-docs:
	@$(REPO_ROOT)/hack/generate-docs-index.sh

.PHONY: generate
generate: generate-code format revendor generate-docs

#################################################################
# Rules related to binary build, docker image build and release #
#################################################################

.PHONY: install
install:
	@EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) ./hack/install.sh

.PHONY: prepare-docker-builder
prepare-docker-builder:
	@$(REPO_ROOT)/hack/prepare-docker-builder.sh

.PHONY: build-docker-images
build-docker-images:
	@echo "Building docker images for version $(EFFECTIVE_VERSION)"
	@docker buildx build --builder $(DOCKER_BUILDER_NAME) --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) --platform linux/amd64 -t $(LANDSCAPER_CONTROLLER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target landscaper-controller .
	@docker buildx build --builder $(DOCKER_BUILDER_NAME) --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) --platform linux/amd64 -t $(LANDSCAPER_WEBHOOKS_SERVER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target landscaper-webhooks-server .
	@docker buildx build --builder $(DOCKER_BUILDER_NAME) --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) --platform linux/amd64 -t $(LANDSCAPER_AGENT_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target landscaper-agent .
	@docker buildx build --builder $(DOCKER_BUILDER_NAME) --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) --platform linux/amd64 -t $(CONTAINER_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target container-deployer-controller .
	@docker buildx build --builder $(DOCKER_BUILDER_NAME) --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) --platform linux/amd64 -t $(CONTAINER_DEPLOYER_INIT_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target container-deployer-init .
	@docker buildx build --builder $(DOCKER_BUILDER_NAME) --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) --platform linux/amd64 -t $(CONTAINER_DEPLOYER_WAIT_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target container-deployer-wait .
	@docker buildx build --builder $(DOCKER_BUILDER_NAME) --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) --platform linux/amd64 -t $(HELM_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target helm-deployer-controller .
	@docker buildx build --builder $(DOCKER_BUILDER_NAME) --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) --platform linux/amd64 -t $(MANIFEST_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target manifest-deployer-controller .
	@docker buildx build --builder $(DOCKER_BUILDER_NAME) --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) --platform linux/amd64 -t $(MOCK_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target mock-deployer-controller .

.PHONY: docker-images
docker-images: prepare-docker-builder build-docker-images

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

.PHONY: install-testcluster-cmd
install-testcluster-cmd:
	@go install $(REPO_ROOT)/hack/testcluster


.PHONY: start-webhooks
start-webhooks:
	@go run $(REPO_ROOT)/cmd/landscaper-webhooks-server -v 3 --kubeconfig=$(KUBECONFIG)
