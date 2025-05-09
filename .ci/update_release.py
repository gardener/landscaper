#!/usr/bin/env python3

import pathlib
from ci.util import (
    check_env,
    ctx,
)
import ccc.github

import os

from github.util import GitHubRepositoryHelper

VERSION_FILE_NAME = 'VERSION'

repo_owner_and_name = check_env('SOURCE_GITHUB_REPO_OWNER_AND_NAME')
repo_dir = check_env('MAIN_REPO_DIR')

repo_owner, repo_name = repo_owner_and_name.split('/')

repo_path = pathlib.Path(repo_dir).resolve()

version_file_path = repo_path / VERSION_FILE_NAME

version_file_contents = version_file_path.read_text()

cfg_factory = ctx().cfg_factory()
github_cfg = cfg_factory.github('github_com')
github_api = ccc.github.github_api(github_cfg)

github_repo_helper = GitHubRepositoryHelper(
    owner=repo_owner,
    name=repo_name,
    github_api=github_api,
)

gh_release = github_repo_helper.repository.release_from_tag(version_file_contents)


try:
    os.environ['INTEGRATION_TEST_PATH']
except KeyError:
    print("No integration test output path found. Output will not be added to release")
else:
    integration_test_path = check_env('INTEGRATION_TEST_PATH')
    integration_test_path = pathlib.Path(integration_test_path).resolve()
    integration_test_path = integration_test_path / "ttt.log"
    gh_release.upload_asset(
        content_type='text/plain',
        name=f'integration-test-ocm-result-{version_file_contents}.txt',
        asset=integration_test_path.open(mode='rb'),
    )
