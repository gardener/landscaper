#!/bin/bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

COMPONENT_DIR="$(dirname $0)/.."
BLUEPRINT_DIR="${COMPONENT_DIR}/blueprint"
OCI_ARTIFACT_REF="eu.gcr.io/gardener-project/landscaper/examples/blueprints/guided-tour/export-token:1.0.0"

landscaper-cli blueprints push "${OCI_ARTIFACT_REF}" "${BLUEPRINT_DIR}"
