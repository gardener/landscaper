# SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

#### BASE ####
FROM gcr.io/distroless/static-debian11:nonroot AS base

#RUN apt install -y --no-cache ca-certificates

#### Landscaper Controller ####
FROM base as landscaper-controller

ARG TARGETOS
ARG TARGETARCH
WORKDIR /
COPY bin/landscaper-controller-$TARGETOS.$TARGETARCH /landscaper-controller
USER 65532:65532

ENTRYPOINT ["/landscaper-controller"]

#### Landsacper webhooks server ####
FROM base as landscaper-webhooks-server

ARG TARGETOS
ARG TARGETARCH
WORKDIR /
COPY bin/landscaper-webhooks-server-$TARGETOS.$TARGETARCH /landscaper-webhooks-server
USER 65532:65532

ENTRYPOINT ["/landscaper-webhooks-server"]

#### Container Deployer Controller ####
FROM base as container-deployer-controller

ARG TARGETOS
ARG TARGETARCH
WORKDIR /
COPY bin/container-deployer-controller-$TARGETOS.$TARGETARCH /container-deployer-controller
USER 65532:65532

ENTRYPOINT ["/container-deployer-controller"]

#### Container Deployer Init ####
FROM base as container-deployer-init

ARG TARGETOS
ARG TARGETARCH
WORKDIR /
COPY bin/container-deployer-init-$TARGETOS.$TARGETARCH /container-deployer-init
USER 65532:65532

ENTRYPOINT ["/container-deployer-init"]

#### Container Deployer wait ####
FROM base as container-deployer-wait

ARG TARGETOS
ARG TARGETARCH
WORKDIR /
COPY bin/container-deployer-wait-$TARGETOS.$TARGETARCH /container-deployer-wait
USER 65532:65532

ENTRYPOINT ["/container-deployer-wait"]

#### Helm Deployer Controller ####
FROM base as helm-deployer-controller

ARG TARGETOS
ARG TARGETARCH
WORKDIR /
COPY bin/helm-deployer-controller-$TARGETOS.$TARGETARCH /helm-deployer-controller
USER 65532:65532

ENTRYPOINT ["/helm-deployer-controller"]

#### Manifest Deployer Controller ####
FROM base as manifest-deployer-controller

ARG TARGETOS
ARG TARGETARCH
WORKDIR /
COPY bin/manifest-deployer-controller-$TARGETOS.$TARGETARCH /manifest-deployer-controller
USER 65532:65532

ENTRYPOINT ["/manifest-deployer-controller"]

#### Mock Deployer Controller ####
FROM base as mock-deployer-controller

ARG TARGETOS
ARG TARGETARCH
WORKDIR /
COPY bin/mock-deployer-controller-$TARGETOS.$TARGETARCH /mock-deployer-controller
USER 65532:65532

ENTRYPOINT ["/mock-deployer-controller"]
