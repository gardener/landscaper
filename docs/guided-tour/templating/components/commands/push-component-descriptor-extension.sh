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

# COMPONENT_DIR is the is the path to the directory that contains the component-descriptor.yaml and resources.yaml
EXAMPLE_DIR="$(dirname $0)/.."
COMPONENT_DIR="${EXAMPLE_DIR}/component-extension"

# TRANSPORT_FILE is the path to the transport tar file that will be created and pushed to the oci registry
TRANSPORT_FILE=${EXAMPLE_DIR}/commands/transport-extension.tar

echo "Component directory: ${COMPONENT_DIR}"
echo "Transport file:      ${TRANSPORT_FILE}"

if [ -f "${TRANSPORT_FILE}" ]; then
  echo "Removing old transport file"
  rm "${TRANSPORT_FILE}"
fi

echo "Creating transport file"
landscaper-cli component-cli component-archive "${COMPONENT_DIR}" "${TRANSPORT_FILE}"

echo "Pushing transport file to oci registry"
component-cli ctf push "${TRANSPORT_FILE}"
