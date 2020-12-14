#!/bin/bash
#
# Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# SPDX-License-Identifier: Apache-2.0

set -e

BLUEPRINT_INGRESS_NGINX_VERSION="v0.2.1"
BLUEPRINT_ECHO_SERVER_VERSION="v0.1.1"
BLUEPRINT_AGGREGATED_VERSION="v0.1.1"

CURRENT_DIR=$(dirname $0)
PROJECT_ROOT="${CURRENT_DIR}"/..

if ! command -v landscaper-cli &> /dev/null
then
    echo "landscaper-cli could not be found"
    echo "install it by running: make install-cli"
    exit
fi

blueprints=(
  "eu.gcr.io/gardener-project/landscaper/tutorials/blueprints/ingress-nginx:${BLUEPRINT_INGRESS_NGINX_VERSION},./docs/tutorials/resources/ingress-nginx/blueprint"
  "eu.gcr.io/gardener-project/landscaper/tutorials/blueprints/echo-server:${BLUEPRINT_ECHO_SERVER_VERSION},./docs/tutorials/resources/echo-server/blueprint"
  "eu.gcr.io/gardener-project/landscaper/tutorials/blueprints/simple-aggregated:${BLUEPRINT_AGGREGATED_VERSION},./docs/tutorials/resources/aggregated/blueprint"
)
component_descriptors=(
  "./docs/tutorials/resources/ingress-nginx"
  "./docs/tutorials/resources/echo-server"
  "./docs/tutorials/resources/aggregated"
)

for i in "${blueprints[@]}"; do
  IFS=',' read ref blueprints_path <<< "${i}"

  landscaper-cli blueprint push ${ref} ${blueprints_path}
done

for i in "${component_descriptors[@]}"; do
  landscaper-cli cd push ${i}
done
