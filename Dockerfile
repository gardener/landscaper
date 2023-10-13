# SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

#### BUILDER ####
FROM golang:1.20.10 AS builder

WORKDIR /go/src/github.com/gardener/landscaper
COPY . .

ARG EFFECTIVE_VERSION

RUN make install EFFECTIVE_VERSION=$EFFECTIVE_VERSION

#### BASE ####
FROM gcr.io/distroless/static-debian11:nonroot AS base

#RUN apt install -y --no-cache ca-certificates

#### Landscaper Controller ####
FROM base as landscaper-controller

COPY --from=builder /go/bin/landscaper-controller /landscaper-controller

WORKDIR /

ENTRYPOINT ["/landscaper-controller"]

#### Landsacper webhooks server ####
FROM base as landscaper-webhooks-server

COPY --from=builder /go/bin/landscaper-webhooks-server /landscaper-webhooks-server

WORKDIR /

ENTRYPOINT ["/landscaper-webhooks-server"]

#### Landscaper Agent ####
FROM base as landscaper-agent

COPY --from=builder /go/bin/landscaper-agent /landscaper-agent

WORKDIR /

ENTRYPOINT ["/landscaper-agent"]

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

#### Manifest Deployer Controller ####
FROM base as manifest-deployer-controller

COPY --from=builder /go/bin/manifest-deployer-controller /manifest-deployer-controller

WORKDIR /

ENTRYPOINT ["/manifest-deployer-controller"]

#### Mock Deployer Controller ####
FROM base as mock-deployer-controller

COPY --from=builder /go/bin/mock-deployer-controller /mock-deployer-controller

WORKDIR /

ENTRYPOINT ["/mock-deployer-controller"]
