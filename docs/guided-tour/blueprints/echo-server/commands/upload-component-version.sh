#!/bin/bash
#
# Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

# Currently either "v2" or "ocm.software/v3alpha1" 
SCHEMA_VERSION=$1
LOCAL_BLUEPRINT=$2

PATH_SUFFIX=""
if [[ $SCHEMA_VERSION == "" ]]; then
    echo "Please provide schema version as argument 1 (v2 or ocm.software/v3alpha1)"
    exit 1
elif [[ $SCHEMA_VERSION == "ocm.software/v3alpha1" ]]; then
    PATH_SUFFIX="v3"
elif [[ $SCHEMA_VERSION == "v2" ]]; then
    PATH_SUFFIX=$SCHEMA_VERSION
else
    echo "invalid argument value ${SCHEMA_VERSION}, please specify either v2 or ocm.software/v3alpha1"
    exit 1
fi

if [[ $LOCAL_BLUEPRINT == "" ]]; then
    echo "Please specify whether the resource shall be added as reference (external) or as local blob (local) as argument 2"
    exit 1
elif [[ $LOCAL_BLUEPRINT == "local" ]] || [[ $LOCAL_BLUEPRINT == "external" ]]; then
    PATH_SUFFIX="${PATH_SUFFIX}-${LOCAL_BLUEPRINT}"
else
    echo "invalid argument value ${LOCAL_BLUEPRINT}, please specify either local or external"
    exit 1
fi

SCRIPT_DIR=$(dirname "$0")
PARENT_DIR="${SCRIPT_DIR}/.."
COMPONENT_DIR="$(realpath "${PARENT_DIR}")/component-archive/${PATH_SUFFIX}"
BLUEPRINT_DIR="$(realpath "${PARENT_DIR}")/blueprint"

if [ -d "${COMPONENT_DIR}" ]; then
  echo "removing old component archive"
  rm -r "${COMPONENT_DIR}"
fi

COMPONENT_NAME="github.com/gardener/landscaper-examples/guided-tour/echo-server"
# version 2.x.x because 1.x.x are reserved for the legacy components created with component-cli and the corresponding examples
COMPONENT_VERSION="2.0.0"
COMPONENT_PROVIDER="internal"

# A Component Archive is a file system representation of a OCM Repository capable of hosting exactly one Component Version
echo "creating component archive at ${COMPONENT_DIR}"
ocm create componentarchive ${COMPONENT_NAME} ${COMPONENT_VERSION} --provider ${COMPONENT_PROVIDER} --file ${COMPONENT_DIR} --scheme ${SCHEMA_VERSION}

# Add the blueprint as a resource to the component version
if [[ $LOCAL_BLUEPRINT == "local" ]]; then
    # Add blueprint as local (also known as inline) resource
    ocm add resource ${COMPONENT_DIR} --type blueprint --name blueprint --version 1.0.0 --inputType dir --inputPath "${BLUEPRINT_DIR}" --inputCompress --mediaType "application/vnd.gardener.landscaper.blueprint.v1+tar+gzip"
elif [[  $LOCAL_BLUEPRINT == "external" ]]; then
    # or, if the blueprint is already uploaded to an oci registry, e.g. with the landscaper-cli 
    # Add the image reference to the blueprint
    ocm add resource ${COMPONENT_DIR} --type blueprint --name blueprint --version 1.0.0 --accessType ociArtifact --reference eu.gcr.io/gardener-project/landscaper/examples/blueprints/guided-tour/echo-server:1.0.0
fi

# Add the helm chart as an external resource to the component version
# Adding resources besides the blueprint as local blob is currently not supported by the landscaper
ocm add resource ${COMPONENT_DIR} --type helmChart --name echo-server-chart --version 1.0.0 --accessType ociArtifact --reference eu.gcr.io/gardener-project/landscaper/examples/charts/guided-tour/echo-server:1.0.0

# Add the docker image as an external resource to the component version
# Adding resources besides the blueprint as local blob is currently not supported by the landscaper
ocm add resource ${COMPONENT_DIR} --type ociImage --name echo-server-image --version 0.2.3 --accessType ociArtifact --reference hashicorp/http-echo:0.2.3


# Transfer the Component Version from the file system representation of an OCM Repository to an oci registry representation of an OCM Repository
# echo "pushing component version to oci registry"
OCM_REPOSITORY="eu.gcr.io/gardener-project/landscaper/examples"
ocm transfer componentarchive --overwrite ${COMPONENT_DIR} ${OCM_REPOSITORY}
