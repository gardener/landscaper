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

# This commands adds the components to a ctf (common transport archive), which is a file system representation of an
# oci registry
# --create specifies that the ctf file/directory should be created if it does not exist yet
# --file specifies the target ctf file/directory where the components should be added
echo "add components"
ocm add components --create --file "$ctf_dir" ${COMPONENT_DIR}/commands/components.yaml

# This command transfers the components contained in the specified ctf to another component repository
# (here, an oci registry)
# --enforce specifies that already existing components in the target should always be overwritten with the ones
# from your source
ocm transfer ctf --overwrite "$ctf_dir" eu.gcr.io/gardener-project/landscaper/examples

## To inspect a specific component, you can use the following command to download into a component archive (a simple file
## system representation of a single component)
## this can be done from a remote repository
# ocm download component eu.gcr.io/gardener-project/landscaper/examples//github.com/gardener/landscaper-examples/guided-tour/echo-server:2.0.0 -O ../archive
## or from a local ctf representation
# ocm download component ../tour-ctf//github.com/gardener/landscaper-examples/guided-tour/echo-server:2.0.0 -O ../archive