#!/bin/bash
#
# SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

COMMAND_DIR="$(dirname $0)"
HACK_DIR="${COMMAND_DIR}/../../../hack"

source "${HACK_DIR}/settings"
"${HACK_DIR}/upload-component.sh" "${COMMAND_DIR}/component-constructor.yaml" "$REPO_BASE_URL_INTEGRATION_TESTS"
