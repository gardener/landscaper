#!/bin/bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

COMPONENT_DIR="$(dirname $0)/.."
TRANSPORT_FILE=${COMPONENT_DIR}/commands/transport.tar

${COMPONENT_DIR}/../../hack/create-and-push-component.sh "${COMPONENT_DIR}" "${TRANSPORT_FILE}"
