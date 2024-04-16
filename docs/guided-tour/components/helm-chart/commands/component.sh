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

ctf_dir=$(mktemp -d)

# This commands adds the components to a ctf (common transport archive), which is a file system representation of an
# oci registry
# --create specifies that the ctf file/directory should be created if it does not exist yet
# --file specifies the target ctf file/directory where the components should be added
echo "add components"
ocm add components --create --file "${ctf_dir}" ${component_dir}/commands/component-constructor.yaml

# This command transfers the components contained in the specified ctf to another component repository
# (here, an oci registry)
# --enforce specifies that already existing components in the target should always be overwritten with the ones
# from your source
ocm transfer ctf --overwrite "${ctf_dir}" "${REPO_BASE_URL}"


## Download
# ocm download component eu.gcr.io/gardener-project/landscaper/examples//github.com/gardener/landscaper-examples/guided-tour/helm-chart:1.0.0 -O ./archive-helm-chart
