# SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

#### BUILDER ####
FROM eu.gcr.io/gardener-project/3rd/golang:1.15.5 AS builder

WORKDIR /go/src/github.com/gardener/landscaper
COPY . .

ARG EFFECTIVE_VERSION

RUN make install EFFECTIVE_VERSION=$EFFECTIVE_VERSION

#### BASE ####
FROM eu.gcr.io/gardenlinux/gardenlinux:184.0 AS base

#### Helm Deployer Controller ####
FROM base as landscaper-controller

COPY --from=builder /go/bin/landscaper-controller /landscaper-controller

WORKDIR /

ENTRYPOINT ["/landscaper-controller"]

#### Container Deployer Controller ####
FROM base as container-deployer-controller

COPY --from=builder /go/bin/container-deployer-controller /container-deployer-controller

WORKDIR /

ENTRYPOINT ["/container-deployer-controller"]

#### Container Deployer Init ####
FROM base as container-deployer-init

COPY --from=builder /go/bin/container-deployer-init /container-deployer-init

WORKDIR /

ENTRYPOINT ["/container-deployer-init"]

#### Container Deployer wait ####
FROM base as container-deployer-wait

COPY --from=builder /go/bin/container-deployer-wait /container-deployer-wait

WORKDIR /

ENTRYPOINT ["/container-deployer-wait"]

#### Helm Deployer Controller ####
FROM base as helm-deployer-controller

COPY --from=builder /go/bin/helm-deployer-controller /helm-deployer-controller

WORKDIR /

ENTRYPOINT ["/helm-deployer-controller"]

