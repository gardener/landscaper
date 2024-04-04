#!/bin/bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors.
#
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

PROJECT_ROOT="$(realpath $(dirname $0)/..)"
mod="$1"

version="$(cat "$PROJECT_ROOT/go.mod" | grep -m 1 "$mod ")" # fetch line containing the version
version=${version%%//*} # remove potential comment at end of line
version=$(sed -r 's@^[[:blank:]]+|[[:blank:]]+$@@g' <<< $version) # remove leading and trailing whitespace
version=${version#$mod' '}

# resolve replace directives
if cat "$PROJECT_ROOT/go.mod" | grep "$mod => $mod" &>/dev/null || cat "$PROJECT_ROOT/go.mod" | grep "$mod $version => $mod" &>/dev/null; then
  version="$(cat "$PROJECT_ROOT/go.mod" | grep -E -m 1 "$mod .*=> $mod")" # fetch line containing the replace
  version=${version%%//*} # remove potential comment at end of line
  version=${version##*'=>'} # remove everything before the '=>'
  version=$(sed -r 's@^[[:blank:]]+|[[:blank:]]+$@@g' <<< $version) # remove leading and trailing whitespace
  version=${version#$mod' '}
fi

if [[ ${NO_PREFIX:-"false"} != "false" ]]; then
  version=${version#v}
fi

echo "$version"