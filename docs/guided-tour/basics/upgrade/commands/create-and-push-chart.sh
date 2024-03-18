#!/bin/bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0


# Get an access token:
# gcloud auth login
# gcloud auth print-access-token
ACCESS_TOKEN=...

COMPONENT_DIR="$(dirname $0)/.."

helm package "${COMPONENT_DIR}/chart/hello-world" -d "${COMPONENT_DIR}/commands"

helm registry login eu.gcr.io -u oauth2accesstoken -p "${ACCESS_TOKEN}"

helm push "${COMPONENT_DIR}/commands/hello-world-1.0.1.tgz" oci://eu.gcr.io/gardener-project/landscaper/examples/charts
