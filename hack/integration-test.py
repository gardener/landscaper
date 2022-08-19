#!/usr/bin/env python3

# SPDX-FileCopyrightText: 2022 "SAP SE or an SAP affiliate company and Gardener contributors"
#
# SPDX-License-Identifier: Apache-2.0

# Helper script executing the integration tests in the context of a Gardener Concours pipeline job with access to the cc-config.
# It mainly fetches the access data of the 'laas' project of the Gardener Canary Landscape and stores it in a file. Then it
# calls the script '/.ci/local-integration-test-with-cluster-creation'.

import os
import sys
import utils
import yaml
import json
import model.container_registry
import oci.auth as oa

from util import ctx
from subprocess import run

version = os.environ["VERSION"]
source_path = os.environ["SOURCE_PATH"]

factory = ctx().cfg_factory()
print("Starting integration tests with version " + version + " in sourcepath " + source_path)

landscape_kubeconfig = factory.kubernetes("landscaper-integration-test")

with utils.TempFileAuto(prefix="landscape_kubeconfig_") as kubeconfig_temp_file:
    kubeconfig_temp_file.write(yaml.safe_dump(landscape_kubeconfig.kubeconfig()))
    landscape_kubeconfig_path = kubeconfig_temp_file.switch()

    command = [source_path + "/.ci/local-integration-test-with-cluster-creation", landscape_kubeconfig_path, "garden-laas ", version]

    print("Executing command")
    run = run(command)

    if run.returncode != 0:
        raise EnvironmentError("Integration test exited with errors")
