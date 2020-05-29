# Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

REPO_ROOT                   := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

.PHONY: install-requirements
install-requirements:
	@go install -mod=vendor $(REPO_ROOT)/vendor/sigs.k8s.io/controller-tools/cmd/controller-gen
	@curl -sfL "https://install.goreleaser.com/github.com/golangci/golangci-lint.sh" | sh -s -- -b $(go env GOPATH)/bin v1.24.0

.PHONY: revendor
revendor:
	@GO111MODULE=on go mod vendor
    @GO111MODULE=on go mod tidy

.PHONY: format
format:
	@$(REPO_ROOT)/hack/format.sh $(REPO_ROOT)/pkg

.PHONY: check
check:
	@$(REPO_ROOT)/hack/check.sh

.PHONY: verify
verify: check format

.PHONY: generate
generate:
	@$(REPO_ROOT)/hack/generate.sh $(REPO_ROOT)/pkg...
	@echo "CRD generator lacks support for openapi defs which should be fixed with this commit https://github.com/kubernetes-sigs/controller-tools/pull/427"
	@controller-gen output:crd:artifacts:config=crd crd:crdVersions=v1 \
		object:headerFile="$(REPO_ROOT)/hack/boilerplate.go.txt" \
		paths="$(REPO_ROOT)/pkg/apis/core/v1alpha1"
