#!/usr/bin/env python3

import os
from subprocess import Popen, PIPE, STDOUT, run
import sys
import shutil
import utils
import yaml

from util import ctx

print("Getting kubeconfig for integration test")
source_path = os.environ['SOURCE_PATH']
root_path = os.environ["ROOT_PATH"]
try:
    # env var is implicitly set by the output dir in case of a release job
    integration_test_path = os.environ["INTEGRATION_TEST_PATH"]
except KeyError:
    print("Integration test env var not set")


factory = ctx().cfg_factory()
landscaper_latest_version = "v0.29.0"
landscaper_latest_kubeconfig = factory.kubernetes("landscaper-latest")
landscaper_latest_kubeconfig_name = "landscaper_latest_kubeconfig"
landscaper_latest_kubeconfig_path = os.path.join(root_path, source_path,
                                         "integration-test",
                                         landscaper_latest_kubeconfig_name)
utils.write_data(landscaper_latest_kubeconfig_path, yaml.dump(
                landscaper_latest_kubeconfig.kubeconfig()))

landscaper_previous_version = "v0.28.0"
landscaper_previous_kubeconfig = factory.kubernetes("landscaper-previous")
landscaper_previous_kubeconfig_name = "landscaper_previous_kubeconfig"
landscaper_previous_kubeconfig_path = os.path.join(root_path, source_path,
                                              "integration-test",
                                              landscaper_previous_kubeconfig_name)
utils.write_data(landscaper_previous_kubeconfig_path, yaml.dump(
                landscaper_previous_kubeconfig.kubeconfig()))


command = ["new-integration-test", landscaper_previous_kubeconfig_path, landscaper_previous_version]

try:
    # check if path var is set
    print(f" Running test command: {command}")
except NameError:
    print(f" Running command after name error: {command}")
    run = run(command)
else:
    output_path = os.path.join(root_path, integration_test_path, "out")

    with Popen(command, stdout=PIPE, stderr=STDOUT, bufsize=1, universal_newlines=True) as run, open(output_path, 'w') as file:
        for line in run.stdout:
            sys.stdout.write(line)
            file.write(line)

if run.returncode != 0:
    raise EnvironmentError("Integration test exited with errors")

# running latest version
command = ["new-integration-test", landscaper_latest_kubeconfig_path, landscaper_latest_version]

try:
    # check if path var is set
    print(f" Running test command: {command}")
except NameError:
    print(f" Running command after name error: {command}")
    run = run(command)
else:
    output_path = os.path.join(root_path, integration_test_path, "out")

    with Popen(command, stdout=PIPE, stderr=STDOUT, bufsize=1, universal_newlines=True) as run, open(output_path, 'w') as file:
        for line in run.stdout:
            sys.stdout.write(line)
            file.write(line)

if run.returncode != 0:
    raise EnvironmentError("Integration test exited with errors")