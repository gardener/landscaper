#!/bin/bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

# This commands adds the components to a ctf (common transport archive), which is a file system representation of a
# oci registry
# --create specifies that the ctf file/directory should be created if it does not exist yet
# --file specifies the target ctf file/directory where the components should be added
ocm add components --create --file ../tour-ctf ../config-files/components.yaml

# This command transfers the components contained in the specified ctf (here tour-ctf) to another component repository
# (here, an oci registry)
# --enforce specifies that already existing components in the target should always be overwritten with the ones
# from your source
ocm transfer ctf --enforce ../tour-ctf eu.gcr.io/gardener-project/landscaper/examples

## to inspect a specific component, you can use the following command to download into a component archive (a simple file
## system representation of a single component)
## this can be done from a remote repository
# ocm download component eu.gcr.io/gardener-project/landscaper/examples//github.com/gardener/landscaper-examples/guided-tour/external-blueprint:2.0.0 -O ../archive
## or from a local ctf representation
# ocm download component ../tour-ctf//github.com/gardener/landscaper-examples/guided-tour/external-blueprint:2.0.0 -O ../archive