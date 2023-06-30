import os
import yaml

version = os.environ["VERSION"]
repo_ctx = os.environ["REPO_CTX"]
component_descriptor_file = os.environ["COMPONENT_DESCRIPTOR_FILE"]

cd = {
    "meta": {
      "schemaVersion": "v2"
    },
    "component": {
      "name": "github.com/gardener/landscaper",
      "version": version,
      "provider": "internal",
      "repositoryContexts": [
        {
          "baseUrl": repo_ctx,
          "componentNameMapping": "urlPath",
          "type": "ociRegistry"
        }
      ],
      "sources": [],
      "resources": [],
      "componentReferences": []
    }
}

with open(component_descriptor_file, "w+") as cd_file:
    cd_file.write(yaml.safe_dump(cd))
