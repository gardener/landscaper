#!/bin/bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

# OCM Repository the component version is pushed to
OCM_REPOSITORY="eu.gcr.io/gardener-project/landscaper/examples"

# Currently either "v2" or "ocm.software/v3alpha1" 
SCHEMA_VERSION=$1
PUSH=$2

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

if [[ $PUSH == "" ]]; then
    echo "Please specify whether the component version shall be pushed to ${OCM_REPOSITORY} (push) or whether this " \
        "step shall be skipped (no-push) as argument 3."
    exit 1
elif [[ $PUSH != "no-push" ]] && [[ $PUSH != "push" ]]; then
    echo "invalid argument value ${NO_PUSH}, please specify either push or no-push"
    exit 1
fi

SCRIPT_DIR=$(dirname "$0")
PARENT_DIR="${SCRIPT_DIR}/.."
COMPONENT_DIR="$(realpath "${PARENT_DIR}")/component-extension/${PATH_SUFFIX}"
BLUEPRINT_DIR="$(realpath "${PARENT_DIR}")/blueprint"

if [ -d "${COMPONENT_DIR}" ]; then
  echo "removing old component archive"
  rm -r "${COMPONENT_DIR}"
fi

COMPONENT_NAME="github.com/gardener/landscaper-examples/guided-tour/templating-components-extension"
# version 2.x.x because 1.x.x are reserved for the legacy components created with component-cli and the corresponding 
# examples
COMPONENT_VERSION="2.0.0"
COMPONENT_PROVIDER="internal"

# A Component Archive is a file system representation of a OCM Repository capable of hosting exactly one Component 
# Version
echo "creating component archive at ${COMPONENT_DIR}"
ocm create componentarchive ${COMPONENT_NAME} ${COMPONENT_VERSION} --provider ${COMPONENT_PROVIDER} \
    --file ${COMPONENT_DIR} --scheme ${SCHEMA_VERSION}

# Add an oci image as an external resource to the component version
# Adding resources besides the blueprint as local blob is currently not supported by the landscaper
# --skip-digest-generation since the images referenced here do not actually exist and are merely added for templating 
# examples
ocm add resource ${COMPONENT_DIR} --skip-digest-generation --type ociImage --name image-c --version 1.0.0 \
    --accessType ociArtifact --reference eu.gcr.io/gardener-project/landscaper/examples/images/image-c:1.0.0 \
    --label landscaper.gardener.cloud/guided-tour/auxiliary='aux-c'

# Add another oci image as an external resource to the component version
# Adding resources besides the blueprint as local blob is currently not supported by the landscaper
# --skip-digest-generation since the images referenced here do not actually exist and are merely added for templating 
# examples
ocm add resource ${COMPONENT_DIR} --skip-digest-generation --type ociImage --name image-d --version 1.0.0 \
    --accessType ociArtifact --reference eu.gcr.io/gardener-project/landscaper/examples/images/image-d:1.0.0 \
    --label landscaper.gardener.cloud/guided-tour/type='type-d'

# Transfer the Component Version from the file system representation of an OCM Repository to an oci registry 
# representation of an OCM Repository echo "pushing component version to oci registry"
if [[ $PUSH == "no-push" ]]; then
    echo "pushig to ${OCM_REPOSITORY} is skipped"
elif [[ $PUSH == "push" ]]; then
    # Transfer the Component Version from the file system representation of an OCM Repository to an oci registry 
    # representation of an OCM Repository echo "pushing component version to oci registry"
    ocm transfer componentarchive --overwrite ${COMPONENT_DIR} ${OCM_REPOSITORY}
fi
