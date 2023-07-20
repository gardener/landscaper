import os
import yaml

version = os.environ["VERSION"]
repo_ctx = os.environ["REPO_CTX"]
component_descriptor_dir = os.environ["COMPONENT_DESCRIPTOR_DIR"]

cd = None

with open(os.path.join(component_descriptor_dir, "base_component_descriptor_v2"), "r") as base_cd_file:
    cd = yaml.safe_load(base_cd_file.read())

cd["component"]["version"] = version
cd["component"]["repositoryContexts"][0]["baseUrl"] = repo_ctx

with open(os.path.join(component_descriptor_dir, "component_descriptor_v2"), "w+") as cd_file:
    cd_file.write(yaml.safe_dump(cd))
