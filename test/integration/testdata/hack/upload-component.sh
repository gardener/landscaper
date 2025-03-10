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

CTF_DIR=$(mktemp -d)

# Add components to a ctf (common transport archive), which is a file system representation of an oci registry.
echo "Add components"
ocm add components --create --file "$CTF_DIR" "$COMPONENT_CONSTRUCTOR_FILE"

# Transfers the components contained in the specified ctf to another component repository (here, an oci registry).
echo "Transfer components"
ocm transfer ctf --overwrite "$CTF_DIR" "$REPO_BASE_URL"
