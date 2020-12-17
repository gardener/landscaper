# SPDX-FileCopyrightText: 2019 SAP SE or an SAP affiliate company and Gardener contributors.
#
# SPDX-License-Identifier: Apache-2.0

REPO_ROOT                                      := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
VERSION                                        := $(shell cat $(REPO_ROOT)/VERSION)
EFFECTIVE_VERSION                              := $(VERSION)-$(shell git rev-parse HEAD)

REGISTRY                                       := eu.gcr.io/gardener-project/landscaper
LANDSCAPER_CONTROLLER_IMAGE_REPOSITORY         := $(REGISTRY)/landscaper-controller
CONTAINER_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY := $(REGISTRY)/container-deployer-controller
CONTAINER_DEPLOYER_INIT_IMAGE_REPOSITORY       := $(REGISTRY)/container-deployer-init
CONTAINER_DEPLOYER_WAIT_IMAGE_REPOSITORY       := $(REGISTRY)/container-deployer-wait
HELM_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY      := $(REGISTRY)/helm-deployer-controller
MANIFEST_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY  := $(REGISTRY)/manifest-deployer-controller


.PHONY: install-requirements
install-requirements:
	@go install -mod=vendor $(REPO_ROOT)/vendor/sigs.k8s.io/controller-tools/cmd/controller-gen
	@go install -mod=vendor $(REPO_ROOT)/vendor/github.com/ahmetb/gen-crd-api-reference-docs
	@go install -mod=vendor $(REPO_ROOT)/vendor/github.com/golang/mock/mockgen
	@$(REPO_ROOT)/hack/install-requirements.sh

.PHONY: revendor
revendor:
	@$(REPO_ROOT)/hack/revendor.sh

.PHONY: format
format:
	@$(REPO_ROOT)/hack/format.sh $(REPO_ROOT)/pkg $(REPO_ROOT)/test $(REPO_ROOT)/cmd

.PHONY: check
check:
	@$(REPO_ROOT)/hack/check.sh --golangci-lint-config=./.golangci.yaml $(REPO_ROOT)/cmd/... $(REPO_ROOT)/pkg/... $(REPO_ROOT)/test/...

.PHONY: test
test:
	@go test -mod=vendor $(REPO_ROOT)/cmd/... $(REPO_ROOT)/pkg/... $(REPO_ROOT)/test/...

.PHONY: verify
verify: check

.PHONY: generate
generate:
	@$(REPO_ROOT)/hack/generate.sh $(REPO_ROOT)/pkg...

#################################################################
# Rules related to binary build, docker image build and release #
#################################################################

.PHONY: install
install:
	@EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) ./hack/install.sh

.PHONY: docker-images
docker-images:
	@echo "Building docker images for version $(EFFECTIVE_VERSION)"
	@docker build -t $(LANDSCAPER_CONTROLLER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target landscaper-controller .
	@docker build -t $(CONTAINER_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target container-deployer-controller .
	@docker build -t $(CONTAINER_DEPLOYER_INIT_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target container-deployer-init .
	@docker build -t $(CONTAINER_DEPLOYER_WAIT_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target container-deployer-wait .
	@docker build -t $(HELM_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target helm-deployer-controller .
	@docker build -t $(MANIFEST_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION) -f Dockerfile --target manifest-deployer-controller .

.PHONY: docker-images
docker-push:
	@echo "Pushing docker images for version $(EFFECTIVE_VERSION) to registry $(REGISTRY)"
	@if ! docker images $(LANDSCAPER_CONTROLLER_IMAGE_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(EFFECTIVE_VERSION); then echo "$(LANDSCAPER_CONTROLLER_IMAGE_REPOSITORY) version $(EFFECTIVE_VERSION) is not yet built. Please run 'make docker-images'"; false; fi
	@if ! docker images $(CONTAINER_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(EFFECTIVE_VERSION); then echo "$(CONTAINER_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY) version $(EFFECTIVE_VERSION) is not yet built. Please run 'make docker-images'"; false; fi
	@if ! docker images $(CONTAINER_DEPLOYER_INIT_IMAGE_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(EFFECTIVE_VERSION); then echo "$(CONTAINER_DEPLOYER_INIT_IMAGE_REPOSITORY) version $(EFFECTIVE_VERSION) is not yet built. Please run 'make docker-images'"; false; fi
	@if ! docker images $(CONTAINER_DEPLOYER_WAIT_IMAGE_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(EFFECTIVE_VERSION); then echo "$(CONTAINER_DEPLOYER_WAIT_IMAGE_REPOSITORY) version $(EFFECTIVE_VERSION) is not yet built. Please run 'make docker-images'"; false; fi
	@if ! docker images $(HELM_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(EFFECTIVE_VERSION); then echo "$(HELM_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY) version $(EFFECTIVE_VERSION) is not yet built. Please run 'make docker-images'"; false; fi
	@if ! docker images $(MANIFEST_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(EFFECTIVE_VERSION); then echo "$(MANIFEST_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY) version $(EFFECTIVE_VERSION) is not yet built. Please run 'make docker-images'"; false; fi
	@docker push $(LANDSCAPER_CONTROLLER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION)
	@docker push $(CONTAINER_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION)
	@docker push $(CONTAINER_DEPLOYER_INIT_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION)
	@docker push $(CONTAINER_DEPLOYER_WAIT_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION)
	@docker push $(HELM_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION)
	@docker push $(MANIFEST_DEPLOYER_CONTROLLER_IMAGE_REPOSITORY):$(EFFECTIVE_VERSION)

######################
# Tutorial resources #
######################

.PHONY: upload-tutorial-resources
upload-tutorial-resources:
	@./hack/upload-tutorial-resources.sh
