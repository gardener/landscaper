package ocmlib

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/gardener/landscaper/apis/config"
	"github.com/mandelsoft/vfs/pkg/osfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/oci/identity"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/runtime"
	"os"
	"path/filepath"

	"github.com/gardener/landscaper/apis/core/v1alpha1"
	. "github.com/open-component-model/ocm/pkg/testutils"
)

const (
	USERNAME = "testuser"
	PASSWORD = "testpassword"
)

var (
	data = `{"repositoryContext":{"baseUrl":"eu.gcr.io/gardener-project/landscaper/examples","type":"ociRegistry"},"componentName":"github.com/gardener/landscaper-examples/guided-tour/helm-chart-resource","version":"1.0.0"}`

	auth1         = base64.StdEncoding.EncodeToString([]byte(`testuser1:testpassword1`))
	auth2         = base64.StdEncoding.EncodeToString([]byte(`testuser2:testpassword2`))
	dockerconfig1 = []byte(fmt.Sprintf(`{"auths": {"ghcr.io": {"auth": "%s"},"https://index.docker.io/v1/": {"auth": "%s"}}}`, auth1, auth1))
	dockerconfig2 = []byte(fmt.Sprintf(`{"auths": {"ghcr.io/test/repo": {"auth": "%s"},"https://index.docker.io/v1/": {"auth": "%s"}}}`, auth2, auth2))
	dockerconfigs = map[string][]byte{"dockerconfig1.json": dockerconfig1, "dockerconfig2.json": dockerconfig2}
)

var _ = Describe("ocm-lib facade implementation", func() {
	ctx := context.Background()

	BeforeEach(func() {
		fs := osfs.New()
		for name, config := range dockerconfigs {
			f := Must(fs.OpenFile(filepath.Join("testdata", name), os.O_CREATE|os.O_RDWR, 0o777))
			_ = Must(f.Write(config))
			f.Close()
		}
	})

	It("get component version from component descriptor reference", func() {
		cdref := &v1alpha1.ComponentDescriptorReference{}
		MustBeSuccessful(runtime.DefaultYAMLEncoding.Unmarshal([]byte(data), &cdref))

		r := &RegistryAccess{
			octx:    ocm.DefaultContext(),
			session: ocm.NewSession(datacontext.NewSession()),
		}

		cv := Must(r.GetComponentVersion(context.Background(), cdref))
		Expect(cv).NotTo(BeNil())
		Expect(cv.GetName()).To(Equal("github.com/gardener/landscaper-examples/guided-tour/helm-chart-resource"))
		Expect(cv.GetVersion()).To(Equal("1.0.0"))
	})
	It("test component version methods", func() {
		cdref := &v1alpha1.ComponentDescriptorReference{}
		MustBeSuccessful(runtime.DefaultYAMLEncoding.Unmarshal([]byte(data), &cdref))

		r := &RegistryAccess{
			octx:    ocm.DefaultContext(),
			session: ocm.NewSession(datacontext.NewSession()),
		}
		cv := Must(r.GetComponentVersion(context.Background(), cdref))

		Expect(cv.GetName()).To(Equal("github.com/gardener/landscaper-examples/guided-tour/helm-chart-resource"))
		Expect(cv.GetVersion()).To(Equal("1.0.0"))
		Expect(Must(cv.GetComponentDescriptor()).GetName()).To(Equal("github.com/gardener/landscaper-examples/guided-tour/helm-chart-resource"))
		Expect(Must(cv.GetComponentDescriptor()).GetVersion()).To(Equal("1.0.0"))
	})
	FIt("credentials", func() {
		f := Factory{}
		r := Must(f.NewRegistryAccess(ctx, nil, nil, nil, &config.OCIConfiguration{
			ConfigFiles: []string{"testdata/dockerconfig1.json", "testdata/dockerconfig2.json"},
		}, nil)).(*RegistryAccess)
		creds := Must(identity.GetCredentials(r.octx, "ghcr.io", "/test/repo"))
		props := creds.Properties()
		_ = props
	})
})
