#!/bin/bash
#
# Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# SPDX-License-Identifier: Apache-2.0

set -e

CURRENT_DIR=$(dirname $0)
PROJECT_ROOT="${CURRENT_DIR}"/..

if ! command -v landscaper-cli &> /dev/null
then
    echo "landscaper-cli could not be found"
    echo "install it by running: make install-cli"
    exit
fi

if ! command -v component-cli &> /dev/null
then
    echo "component-cli could not be found"
    echo "install it by running: make install-cli"
    exit
fi

# this array consists of <oci-path of blueprint>;<local path of blueprint>;<default version tag>
# the version tag can be set inline in the blueprint.yaml file itself by the comment line
# # TUTORIAL_BLUEPRINT_VERSION: <my version>
# this is the default that we fall back on in case the version information is not maintained
# in the blueprint file
blueprints=(
  "eu.gcr.io/gardener-project/landscaper/tutorials/blueprints/ingress-nginx;./docs/tutorials/resources/ingress-nginx/blueprint;v0.2.1"
  "eu.gcr.io/gardener-project/landscaper/tutorials/blueprints/echo-server;./docs/tutorials/resources/echo-server/blueprint;v0.1.1"
  "eu.gcr.io/gardener-project/landscaper/tutorials/blueprints/simple-aggregated;./docs/tutorials/resources/aggregated/blueprint;v0.1.1"
)
component_descriptors=(
  "./docs/tutorials/resources/ingress-nginx"
  "./docs/tutorials/resources/echo-server"
  "./docs/tutorials/resources/aggregated"
  "./docs/tutorials/resources/local-ingress-nginx"
  "./docs/tutorials/resources/external-jsonschema/definitions"
  "./docs/tutorials/resources/external-jsonschema/echo-server"
)

function prepare_local_nginx_resources() {
  cp ./docs/tutorials/resources/local-ingress-nginx/component-descriptor.yaml ./docs/tutorials/resources/local-ingress-nginx/component-descriptor.yam_
  component-cli ca resources add ./docs/tutorials/resources/local-ingress-nginx  ./docs/tutorials/resources/local-ingress-nginx/helm-resource.yaml
  component-cli ca resources add ./docs/tutorials/resources/local-ingress-nginx  ./docs/tutorials/resources/local-ingress-nginx/blueprint-resource.yaml
}

function prepare_external_json_schema_resourcess() {
  cp ./docs/tutorials/resources/external-jsonschema/echo-server/component-descriptor.yaml ./docs/tutorials/resources/external-jsonschema/echo-server/component-descriptor.yam_
  component-cli ca resources add ./docs/tutorials/resources/external-jsonschema/echo-server  ./docs/tutorials/resources/external-jsonschema/echo-server/blueprint-resource.yaml
  cp ./docs/tutorials/resources/external-jsonschema/definitions/component-descriptor.yaml ./docs/tutorials/resources/external-jsonschema/definitions/component-descriptor.yam_
  component-cli ca resources add ./docs/tutorials/resources/external-jsonschema/definitions  ./docs/tutorials/resources/external-jsonschema/definitions/jsonschema-resource.yaml
}

function cleanup_local_nginx_resources() {
  if [ -d ./docs/tutorials/resources/local-ingress-nginx/blobs ]; then
    rm -rf ./docs/tutorials/resources/local-ingress-nginx/blobs
  fi
  mv -f ./docs/tutorials/resources/local-ingress-nginx/component-descriptor.yam_ ./docs/tutorials/resources/local-ingress-nginx/component-descriptor.yaml
}

function cleanup_external_json_schema_resourcess() {
  if [ -d ./docs/tutorials/resources/external-jsonschema/echo-server/blobs ]; then
    rm -rf ./docs/tutorials/resources/external-jsonschema/echo-server/blobs
  fi
  mv -f ./docs/tutorials/resources/external-jsonschema/echo-server/component-descriptor.yam_ ./docs/tutorials/resources/external-jsonschema/echo-server/component-descriptor.yaml

  if [ -d ./docs/tutorials/resources/external-jsonschema/definitions/blobs ]; then
    rm -rf ./docs/tutorials/resources/external-jsonschema/definitions/blobs
  fi
  mv -f ./docs/tutorials/resources/external-jsonschema/definitions/component-descriptor.yam_ ./docs/tutorials/resources/external-jsonschema/definitions/component-descriptor.yaml
}

prepare_local_nginx_resources
prepare_external_json_schema_resourcess

for i in "${blueprints[@]}"; do
  IFS=';' read ref blueprints_path version <<< "${i}"

  set +e
  inline_version_string=$(grep "^# TUTORIAL_BLUEPRINT_VERSION" $blueprints_path/blueprint.yaml)
  if [ -n "$inline_version_string" ]; then
    version=${inline_version_string#*:}
  fi
  set -e

  landscaper-cli blueprint push ${ref}:${version// /} ${blueprints_path}
done
for i in "${component_descriptors[@]}"; do
  landscaper-cli component-cli ca remote push $i
done

cleanup_local_nginx_resources
cleanup_external_json_schema_resourcess