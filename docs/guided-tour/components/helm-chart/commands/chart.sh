#!/bin/bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

# Get an access token:
# gcloud auth login
# gcloud auth print-access-token
ACCESS_TOKEN=<your access token>

COMPONENT_DIR="$(dirname $0)/.."

helm package "${COMPONENT_DIR}/chart/echo-server" -d "${COMPONENT_DIR}/commands"

helm registry login eu.gcr.io -u oauth2accesstoken -p "${ACCESS_TOKEN}"

helm push "${COMPONENT_DIR}/commands/echo-server-1.0.0.tgz" oci://eu.gcr.io/gardener-project/landscaper/examples/charts/guided-tour
