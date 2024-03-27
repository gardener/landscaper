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
echo "Component directory: " ${component_dir}

if [[ -z "${repository_base_url}" ]]; then
  echo "Variable repository_base_url must contain the base path of the ocm repository"
  exit 1
fi

ctf_dir=`mktemp -d`

# Add components to a ctf (common transport archive), which is a file system representation of an oci registry.
echo "Add components"
ocm add components --create --file $ctf_dir ./components.yaml

# Transfers the components contained in the specified ctf to another component repository (here, an oci registry).
echo "Transfer components"
ocm transfer ctf --enforce $ctf_dir ${repository_base_url}
