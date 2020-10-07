#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

SOURCE_PATH="$(dirname $0)/.."

cp "${COMPONENT_DESCRIPTOR_DIR}/base_component_descriptor_v2" "${COMPONENT_DESCRIPTOR_PATH}"
