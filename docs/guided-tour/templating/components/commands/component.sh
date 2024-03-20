#!/bin/bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -x

component_dir="$(dirname $0)/.."
cd "${component_dir}"
component_dir="$(pwd)"
echo "component directory: ${component_dir}"

source "${component_dir}/commands/settings"

temp_dir=$(mktemp -d)

# The present example deals with three component versions (root, core, extension), which are created by this script.

# Create core component with two oci images as external resources.
# We skip the digest generation for the resources, because they do not actually exist and are merely added for templating examples.
component_name="github.com/gardener/landscaper-examples/guided-tour/templating-components-core"
component_version="2.2.0"
component_provider="internal"
ca_dir="${temp_dir}/core-component"

echo "creating component archive for core component"
ocm create componentarchive ${component_name} ${component_version} --provider ${component_provider} \
    --file "${ca_dir}" --scheme "v2"

ocm add resource "${ca_dir}" --skip-digest-generation --type ociImage --name image-a --version 1.0.0 \
    --accessType ociArtifact --reference eu.gcr.io/gardener-project/landscaper/examples/images/image-a:1.0.0 \
    --label landscaper.gardener.cloud/guided-tour/type='type-a'

ocm add resource "${ca_dir}" --skip-digest-generation --type ociImage --name image-b --version 1.0.0 \
    --accessType ociArtifact --reference eu.gcr.io/gardener-project/landscaper/examples/images/image-b:1.0.0 \
    --label landscaper.gardener.cloud/guided-tour/auxiliary='aux-b'

echo "transferring core component"
ocm transfer componentarchive --overwrite "${ca_dir}" "${REPO_BASE_URL}"


# Create extension component with two oci images as external resources.
# We skip the digest generation for the resources, because they do not actually exist and are merely added for templating examples.
component_name="github.com/gardener/landscaper-examples/guided-tour/templating-components-extension"
component_version="2.2.0"
component_provider="internal"
ca_dir="${temp_dir}/extension-component"

echo "creating component archive for extension component"
ocm create componentarchive ${component_name} ${component_version} --provider ${component_provider} \
    --file "${ca_dir}" --scheme "v2"

ocm add resource "${ca_dir}" --skip-digest-generation --type ociImage --name image-c --version 1.0.0 \
    --accessType ociArtifact --reference eu.gcr.io/gardener-project/landscaper/examples/images/image-c:1.0.0 \
    --label landscaper.gardener.cloud/guided-tour/auxiliary='aux-c'

ocm add resource "${ca_dir}" --skip-digest-generation --type ociImage --name image-d --version 1.0.0 \
    --accessType ociArtifact --reference eu.gcr.io/gardener-project/landscaper/examples/images/image-d:1.0.0 \
    --label landscaper.gardener.cloud/guided-tour/type='type-d'

echo "transferring extension component"
ocm transfer componentarchive --overwrite "${ca_dir}" "${REPO_BASE_URL}"


# Create root component
ctf_dir="${temp_dir}/root-component"

echo "creating ctf for root component"
ocm add components --create --file "${ctf_dir}" "${component_dir}/commands/component-constructor.yaml"

echo "transferring root component"
ocm transfer ctf --overwrite "${ctf_dir}" "${REPO_BASE_URL}"


## Download
# ocm download component eu.gcr.io/gardener-project/landscaper/examples//github.com/gardener/landscaper-examples/guided-tour/templating-components-root:2.2.0 -O ./archive-root
# ocm download component eu.gcr.io/gardener-project/landscaper/examples//github.com/gardener/landscaper-examples/guided-tour/templating-components-core:2.2.0 -O ./archive-core
# ocm download component eu.gcr.io/gardener-project/landscaper/examples//github.com/gardener/landscaper-examples/guided-tour/templating-components-extension:2.2.0 -O ./archive-ext
