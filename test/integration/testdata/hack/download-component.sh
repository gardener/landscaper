#!/bin/bash
#
# SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -x

# COMPONENT_CONSTRUCTOR_FILE is the is the path to the component-constructor.yaml file.
COMPONENT_CONSTRUCTOR_FILE=$1

REPO_BASE_URL=$2

function print_usage_and_exit {
  >&2 echo "Wrong number of arguments:"
  >&2 echo "  USAGE: $0 COMPONENT_CONSTRUCTOR_FILE REPO_BASE_URL"
  exit 1
}

if [ -z "$COMPONENT_CONSTRUCTOR_FILE" ]; then
  print_usage_and_exit
fi

if [ -z "$REPO_BASE_URL" ]; then
  print_usage_and_exit
fi

echo "Component constructor file: $COMPONENT_CONSTRUCTOR_FILE"
echo "Repository base url:        $REPO_BASE_URL"

OUTPUT_DIR="${HOME}/temp/component"

COMPONENT_NAME=$(yq -r '.components[0].name' "$COMPONENT_CONSTRUCTOR_FILE")
COMPONENT_VERSION=$(yq -r '.components[0].version' "$COMPONENT_CONSTRUCTOR_FILE")

echo "Component name:             ${COMPONENT_NAME}"
echo "Component version:          ${COMPONENT_VERSION}"

ocm download component "${REPO_BASE_URL}//${COMPONENT_NAME}:${COMPONENT_VERSION}" -O "$OUTPUT_DIR"

echo "Component downloaded to: $OUTPUT_DIR"
