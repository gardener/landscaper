# SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
#
# SPDX-License-Identifier: Apache-2.0

import sys
import os
import yaml

from kubernetes import client, config

def main(argv=None):
    imports_path = os.environ["IMPORTS_PATH"]
    exports_path = os.environ["EXPORTS_PATH"]
    component_descriptor_path = os.environ["COMPONENT_DESCRIPTOR_PATH"]
    content_path = os.environ["CONTENT_PATH"]
    state_path = os.environ["STATE_PATH"]
    operation = os.environ["OPERATION"]

    print(f"imports_path={imports_path}")
    print(f"exports_path={exports_path}")
    print(f"component_descriptor_path={component_descriptor_path}")
    print(f"content_path={content_path}")
    print(f"state_path={state_path}")
    print(f"operation={operation}")

    with open(component_descriptor_path, 'r') as f:
        data=f.read()
        print(data)
        components = yaml.safe_load(data)

    exports = components

    with open(exports_path, 'w') as f:
        f.write(yaml.safe_dump(exports))

    return 0

if __name__ == "__main__":
    sys.exit(main())
