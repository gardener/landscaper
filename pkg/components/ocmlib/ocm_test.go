// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package ocmlib_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	corev1 "k8s.io/api/core/v1"

	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/pkg/components/ocmlib"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/open-component-model/ocm/pkg/contexts/oci/identity"
	"github.com/open-component-model/ocm/pkg/runtime"

	"github.com/gardener/landscaper/apis/config"

	. "github.com/open-component-model/ocm/pkg/testutils"

	"github.com/gardener/landscaper/apis/core/v1alpha1"
)

const (
	LOCALCNUDIEREPOPATH  = "../testdata/localcnudierepo"
	LOCALOCMREPOPATH     = "../testdata/localocmrepo"
	COMPDESC_V2_FILENAME = "component-descriptor-v2.yaml"
	COMPDESC_V3_FILENAME = "component-descriptor-v3.yaml"

	USERNAME  = "testuser1"
	PASSWORD  = "testpassword1"
	HOSTNAME1 = "ghcr.io"
	HOSTNAME2 = "https://index.docker.io/v1/"
)

// Prepare Test Data
var (
	componentReference = `
{
  "repositoryContext": {
    "type": "local",
    "filePath": "./"
  },
  "componentName": "example.com/landscaper-component",
  "version": "1.0.0"
}
`

	auth             = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", USERNAME, PASSWORD)))
	dockerconfigdata = []byte(fmt.Sprintf(`
{
       "auths": {
               "%s": {"auth": "%s"},
               "%s": {"auth": "%s"}
       }
}
`, HOSTNAME1, auth, HOSTNAME2, auth))

	ocmconfigdata = []byte(fmt.Sprintf(`
{
  "type": "credentials.config.ocm.software",
  "consumers": [
    {
      "identity": {
        "type": "OCIRegistry",
        "hostname": "%s"
      },
      "credentials": [
        {
          "type": "Credentials",
          "properties": {
            "username": "%s",
            "password": "%s"
          }
        }
      ]
    }
  ]
}
`, HOSTNAME1, USERNAME, PASSWORD))
)

var _ = Describe("ocm-lib facade implementation", func() {
	ctx := context.Background()
	factory := ocmlib.Factory{}

	It("get component version from component descriptor reference (from local repository)", func() {
		// as this test uses the local repository implementation, it tests that the ocmlib-facade's GetComponentVersion
		// method can deal with the legacy ComponentDescriptorReference type rather than testing ocmlib functionality
		cdref := &v1alpha1.ComponentDescriptorReference{}
		MustBeSuccessful(runtime.DefaultYAMLEncoding.Unmarshal([]byte(componentReference), &cdref))
		r := Must(factory.NewRegistryAccess(ctx, nil, nil, nil, &config.LocalRegistryConfiguration{RootPath: LOCALCNUDIEREPOPATH}, nil, nil, nil))

		cv := Must(r.GetComponentVersion(ctx, cdref))
		Expect(cv).NotTo(BeNil())
	})

	It("get component descriptor with v2 as input", func() {
		// check that the component descriptor is not altered by the ocmlib-facade
		compdesc := &types.ComponentDescriptor{}
		compdescData := Must(vfs.ReadFile(osfs.New(), filepath.Join(LOCALCNUDIEREPOPATH, COMPDESC_V2_FILENAME)))
		Expect(runtime.DefaultYAMLEncoding.Unmarshal(compdescData, compdesc)).To(Succeed())

		cdref := &v1alpha1.ComponentDescriptorReference{}
		MustBeSuccessful(runtime.DefaultYAMLEncoding.Unmarshal([]byte(componentReference), &cdref))
		r := Must(factory.NewRegistryAccess(ctx, nil, nil, nil, &config.LocalRegistryConfiguration{RootPath: LOCALCNUDIEREPOPATH}, nil, nil, nil))
		cv := Must(r.GetComponentVersion(ctx, cdref))

		Expect(reflect.DeepEqual(cv.GetComponentDescriptor(), compdesc)).To(BeTrue())
	})

	It("get component descriptor with v3 as input", func() {
		// component-descriptor-v2 and component-descriptor-v3 describe identical components with different versions ocm
		// versions and the ocmlib-facade should decode even the v3 version correctly into the landscapers' internal
		// v2 representation
		compdesc := &types.ComponentDescriptor{}
		compdescData := Must(vfs.ReadFile(osfs.New(), filepath.Join(LOCALCNUDIEREPOPATH, COMPDESC_V2_FILENAME)))
		Expect(runtime.DefaultYAMLEncoding.Unmarshal(compdescData, compdesc)).To(Succeed())

		cdref := &v1alpha1.ComponentDescriptorReference{}
		MustBeSuccessful(runtime.DefaultYAMLEncoding.Unmarshal([]byte(componentReference), &cdref))
		r := Must(factory.NewRegistryAccess(ctx, nil, nil, nil, &config.LocalRegistryConfiguration{RootPath: LOCALOCMREPOPATH}, nil, nil, nil))
		cv := Must(r.GetComponentVersion(ctx, cdref))

		Expect(reflect.DeepEqual(cv.GetComponentDescriptor(), compdesc)).To(BeTrue())
	})

	It("dockerconfig credentials from filesystem", func() {
		// Prepare memory test filesystem with dockerconfig credentials
		fs := memoryfs.New()
		Expect(fs.MkdirAll("testdata", 0o777)).To(Succeed())
		dockerconfigs := map[string][]byte{"dockerconfig.json": dockerconfigdata}
		for name, config := range dockerconfigs {
			f := Must(fs.OpenFile(filepath.Join("testdata", name), os.O_CREATE|os.O_RDWR, 0o777))
			_ = Must(f.Write(config))
			Expect(f.Close()).To(Succeed())
		}
		// Create a Registry Access and check whether credentials are properly set and can be found
		r := Must(factory.NewRegistryAccess(ctx, fs, nil, nil, nil, &config.OCIConfiguration{
			ConfigFiles: []string{"testdata/dockerconfig.json"},
		}, nil)).(*ocmlib.RegistryAccess)
		creds := Must(identity.GetCredentials(r.OCMContext(), "ghcr.io", "/test/repo"))
		props := creds.Properties()
		Expect(props["username"]).To(Equal(USERNAME))
		Expect(props["password"]).To(Equal(PASSWORD))
	})

	It("dockerconfig credentials from secrets", func() {
		// Prepare secret with dockerconfig credentials
		secrets := []corev1.Secret{{
			Data: map[string][]byte{corev1.DockerConfigJsonKey: dockerconfigdata},
		}}
		// Create a Registry Access and check whether credentials are properly set and can be found
		r := Must(factory.NewRegistryAccess(ctx, nil, secrets, nil, nil, nil, nil)).(*ocmlib.RegistryAccess)
		creds := Must(identity.GetCredentials(r.OCMContext(), "ghcr.io", "/test/repo"))
		props := creds.Properties()
		Expect(props["username"]).To(Equal(USERNAME))
		Expect(props["password"]).To(Equal(PASSWORD))
	})

	It("ocm credentials from secrets", func() {
		// Prepare secret with ocmconfig credentials
		secrets := []corev1.Secret{{
			Data: map[string][]byte{".ocmcredentialconfig": ocmconfigdata},
		}}
		r := Must(factory.NewRegistryAccess(ctx, nil, secrets, nil, nil, nil, nil)).(*ocmlib.RegistryAccess)
		creds := Must(identity.GetCredentials(r.OCMContext(), HOSTNAME1, "/test/repo"))
		props := creds.Properties()
		Expect(props["username"]).To(Equal(USERNAME))
		Expect(props["password"]).To(Equal(PASSWORD))
	})

	It("blueprint from inline component descriptor with single inline component and blob file system", func() {
		inlineComponentReference := `
repositoryContext:
  type: inline
  compDescDirPath: /
  blobDirPath: /blobs
  fileSystem: 
    component-descriptor1.yaml: |
      meta:
        schemaVersion: v2

      component:
        name: example.com/landscaper-component
        version: 1.0.0

        provider: internal

        repositoryContexts:
        - type: ociRegistry
          baseUrl: "/"

        sources: []
        componentReferences: []

        resources:
        - name: blueprint
          type: blueprint
          version: 1.0.0
          relation: local
          access:
            type: localFilesystemBlob
            mediaType: application/vnd.gardener.landscaper.blueprint.v1+tar+gzip
            filename: blueprint
    blobs:
      blueprint:
        blueprint.yaml: |
          apiVersion: landscaper.gardener.cloud/v1alpha1
          kind: Blueprint

          annotations:
            local/name: root-a
            local/version: 1.0.0

          imports:
          - name: imp-a
            type: data
            schema:
              type: string

          exports:
          - name: exp-a
            type: data
            schema:
              type: string

          deployExecutions:
          - type: GoTemplate
            template: |
              deployItems:
              - name: main
                type: landscaper.gardener.cloud/mock
                config:
                apiVersion: mock.deployer.landscaper.gardener.cloud/v1alpha1
                  kind: ProviderConfiguration
                  providerStatus:
                    apiVersion: mock.deployer.landscaper.gardener.cloud/v1alpha1
                    kind: ProviderStatus
                    imp: {{ index .imports "imp-a" }}
                  export:
                    exp-a: exp-mock

          exportExecutions:
          - type: GoTemplate
            template: |
              exports:
                exp-a: {{ index .values.deployitems.main "exp-a" }}

componentName: example.com/landscaper-component
version: 1.0.0
`
		cdref := &v1alpha1.ComponentDescriptorReference{}
		MustBeSuccessful(runtime.DefaultYAMLEncoding.Unmarshal([]byte(inlineComponentReference), &cdref))
		r := Must(factory.NewRegistryAccess(ctx, nil, nil, nil, &config.LocalRegistryConfiguration{RootPath: LOCALCNUDIEREPOPATH}, nil, nil, nil))
		cv := Must(r.GetComponentVersion(ctx, cdref))
		Expect(cv).NotTo(BeNil())
		res := Must(cv.GetResource("blueprint", nil))
		content := Must(res.GetTypedContent(ctx))
		bp, ok := content.Resource.(*blueprints.Blueprint)
		Expect(ok).To(BeTrue())
		_ = bp
	})

	It("blueprint from inline component descriptor with separate inline component and blob file system", func() {
		inlineComponentReference := `
repositoryContext:
  type: inline
  fileSystem: 
    component-descriptor1.yaml: |
      meta:
        schemaVersion: v2

      component:
        name: example.com/landscaper-component
        version: 1.0.0

        provider: internal

        repositoryContexts:
        - type: ociRegistry
          baseUrl: "/"

        sources: []
        componentReferences: []

        resources:
        - name: blueprint
          type: blueprint
          version: 1.0.0
          relation: local
          access:
            type: localFilesystemBlob
            mediaType: application/vnd.gardener.landscaper.blueprint.v1+tar+gzip
            filename: blueprint
  blobFs:
    blueprint:
      blueprint.yaml: |
        apiVersion: landscaper.gardener.cloud/v1alpha1
        kind: Blueprint

        annotations:
          local/name: root-a
          local/version: 1.0.0

        imports:
        - name: imp-a
          type: data
          schema:
            type: string

        exports:
        - name: exp-a
          type: data
          schema:
            type: string

        deployExecutions:
        - type: GoTemplate
          template: |
            deployItems:
            - name: main
              type: landscaper.gardener.cloud/mock
              config:
              apiVersion: mock.deployer.landscaper.gardener.cloud/v1alpha1
                kind: ProviderConfiguration
                providerStatus:
                  apiVersion: mock.deployer.landscaper.gardener.cloud/v1alpha1
                  kind: ProviderStatus
                  imp: {{ index .imports "imp-a" }}
                export:
                  exp-a: exp-mock

        exportExecutions:
        - type: GoTemplate
          template: |
            exports:
              exp-a: {{ index .values.deployitems.main "exp-a" }}

componentName: example.com/landscaper-component
version: 1.0.0
`
		cdref := &v1alpha1.ComponentDescriptorReference{}
		MustBeSuccessful(runtime.DefaultYAMLEncoding.Unmarshal([]byte(inlineComponentReference), &cdref))
		r := Must(factory.NewRegistryAccess(ctx, nil, nil, nil, &config.LocalRegistryConfiguration{RootPath: LOCALCNUDIEREPOPATH}, nil, nil, nil))
		cv := Must(r.GetComponentVersion(ctx, cdref))
		Expect(cv).NotTo(BeNil())
		res := Must(cv.GetResource("blueprint", nil))
		content := Must(res.GetTypedContent(ctx))
		bp, ok := content.Resource.(*blueprints.Blueprint)
		Expect(ok).To(BeTrue())
		_ = bp
	})
})
