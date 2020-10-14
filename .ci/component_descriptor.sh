#!/bin/bash

# SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

SOURCE_PATH="$(dirname $0)/.."

cp "${COMPONENT_DESCRIPTOR_DIR}/base_component_descriptor_v2" "${COMPONENT_DESCRIPTOR_PATH}"
