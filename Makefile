# SPDX-FileCopyrightText: 2019 SAP SE or an SAP affiliate company and Gardener contributors.
#
# SPDX-License-Identifier: Apache-2.0

REPO_ROOT                                      := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
VERSION                                        := $(shell cat $(REPO_ROOT)/VERSION)
EFFECTIVE_VERSION                              := $(VERSION)-$(shell git rev-parse HEAD)

LANDSCAPER_CONTROLLER_IMAGE_REPOSITORY         := landscaper-controller
LANDSCAPER_WEBHOOKS_SERVER_IMAGE_REPOSITORY    := landscaper-webhooks-server
LANDSCAPER_AGENT_IMAGE_REPOSITORY              := landscaper-agent
CONTAINER_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY := container-deployer-controller
CONTAINER_DEPLOYER_INIT_IMAGE_REPOSITORY       := container-deployer-init
CONTAINER_DEPLOYER_WAIT_IMAGE_REPOSITORY       := container-deployer-wait
HELM_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY      := helm-deployer-controller
MANIFEST_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY  := manifest-deployer-controller
MOCK_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY      := mock-deployer-controller

DOCKER_BUILDER_NAME := "ls-multiarch"
DOCKER_PLATFORM := "linux/amd64"

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
	@$(REPO_ROOT)/hack/local-integration-test $(KUBECONFIG_PATH) $(EFFECTIVE_VERSION) $(USE_OCM_LIB)

.PHONY: integration-test-pure
integration-test-pure:
	@$(REPO_ROOT)/hack/local-integration-test-pure $(KUBECONFIG_PATH) $(EFFECTIVE_VERSION)

.PHONY: integration-test-with-cluster-creation
integration-test-with-cluster-creation:
	@$(REPO_ROOT)/hack/local-integration-test-with-cluster-creation $(KUBECONFIG_PATH) garden-laas $(EFFECTIVE_VERSION) 0 $(USE_OCM_LIB)

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

.PHONY: docker-images
docker-images:
	@$(REPO_ROOT)/hack/prepare-docker-builder.sh
	@echo "Building docker images for version $(EFFECTIVE_VERSION)"
	@docker buildx build --builder $(DOCKER_BUILDER_NAME) --load --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) --platform $(DOCKER_PLATFORM) -t $(LANDSCAPER_CONTROLLER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target landscaper-controller .
	@docker buildx build --builder $(DOCKER_BUILDER_NAME) --load --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) --platform $(DOCKER_PLATFORM) -t $(LANDSCAPER_WEBHOOKS_SERVER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target landscaper-webhooks-server .
	@docker buildx build --builder $(DOCKER_BUILDER_NAME) --load --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) --platform $(DOCKER_PLATFORM) -t $(LANDSCAPER_AGENT_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target landscaper-agent .
	@docker buildx build --builder $(DOCKER_BUILDER_NAME) --load --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) --platform $(DOCKER_PLATFORM) -t $(CONTAINER_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target container-deployer-controller .
	@docker buildx build --builder $(DOCKER_BUILDER_NAME) --load --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) --platform $(DOCKER_PLATFORM) -t $(CONTAINER_DEPLOYER_INIT_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target container-deployer-init .
	@docker buildx build --builder $(DOCKER_BUILDER_NAME) --load --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) --platform $(DOCKER_PLATFORM) -t $(CONTAINER_DEPLOYER_WAIT_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target container-deployer-wait .
	@docker buildx build --builder $(DOCKER_BUILDER_NAME) --load --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) --platform $(DOCKER_PLATFORM) -t $(HELM_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target helm-deployer-controller .
	@docker buildx build --builder $(DOCKER_BUILDER_NAME) --load --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) --platform $(DOCKER_PLATFORM) -t $(MANIFEST_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target manifest-deployer-controller .
	@docker buildx build --builder $(DOCKER_BUILDER_NAME) --load --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) --platform $(DOCKER_PLATFORM) -t $(MOCK_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target mock-deployer-controller .

.PHONY: docker-all
docker-all: docker-images docker-push

.PHONY: component
component:
	@$(REPO_ROOT)/hack/generate-cd.sh

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
