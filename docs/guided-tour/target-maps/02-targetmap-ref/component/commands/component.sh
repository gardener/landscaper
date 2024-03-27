#!/bin/bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -x

COMPONENT_DIR="$(dirname $0)/.."
cd "${COMPONENT_DIR}"
COMPONENT_DIR="$(pwd)"
echo compdir ${COMPONENT_DIR}

source ${COMPONENT_DIR}/commands/settings

ctf_dir=`mktemp -d`

# This commands adds the components to a ctf (common transport archive), which is a file system representation of a
# oci registry
# --create specifies that the ctf file/directory should be created if it does not exist yet
# --file specifies the target ctf file/directory where the components should be added
echo "add components"
ocm add components --create --file $ctf_dir ./components.yaml

# This command transfers the components contained in the specified ctf (here tour-ctf) to another component repository
# (here, an oci registry)
# --enforce specifies that already existing components in the target should always be overwritten with the ones
# from your source
ocm transfer ctf --enforce $ctf_dir ${REPO_BASE_URL}

## to inspect a specific component, you can use the following command to download into a component archive (a simple file
## system representation of a single component)
## this can be done from a remote repository
# ocm download component ${REPO_BASE_URL}/<component name>:<component version e.g. 1.1.0> -O ../archive
