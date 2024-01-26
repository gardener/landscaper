#!/bin/bash
#
# Copyright (c) 2024 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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
