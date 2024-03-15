#!/bin/bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors.
#
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

PROJECT_ROOT="$(realpath $(dirname $0)/..)"
mod="$1"

version="$(cat "$PROJECT_ROOT/go.mod" | grep "$mod ")" # fetch line containing the version
version=${version%%//*} # remove potential comment at end of line
version=$(sed -r 's@^[[:blank:]]+|[[:blank:]]+$@@g' <<< $version) # remove leading and trailing whitespace
version=${version#$mod' '}

echo "$version"