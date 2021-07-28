module github.com/gardener/landscaper

go 1.16

require (
	github.com/Masterminds/sprig/v3 v3.2.2
	github.com/ahmetb/gen-crd-api-reference-docs v0.2.0
	github.com/cloudfoundry-incubator/candiedyaml v0.0.0-20170901234223-a41693b7b7af // indirect
	github.com/containerd/containerd v1.4.4
	github.com/docker/cli v20.10.5+incompatible
	github.com/gardener/component-cli v0.28.0
	github.com/gardener/component-spec/bindings-go v0.0.52
	github.com/gardener/image-vector v0.5.0
	github.com/gardener/landscaper/apis v0.0.0-00010101000000-000000000000
	github.com/go-logr/logr v0.4.0
	github.com/go-logr/zapr v0.4.0
	github.com/golang/mock v1.5.0
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/google/uuid v1.1.2
	github.com/hashicorp/go-multierror v1.1.0
	github.com/mandelsoft/spiff v1.5.0
	github.com/mandelsoft/vfs v0.0.0-20210530103237-5249dc39ce91
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/spf13/cobra v1.2.0
	github.com/spf13/pflag v1.0.5
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415
	github.com/xeipuuv/gojsonschema v1.2.0
	go.uber.org/zap v1.17.0
	golang.org/x/crypto v0.0.0-20210220033148-5ea612d1eb83
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/tools v0.1.3 // indirect
	helm.sh/helm/v3 v3.6.1
	k8s.io/api v0.21.2
	k8s.io/apiextensions-apiserver v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v0.21.2
	k8s.io/utils v0.0.0-20210527160623-6fdb442a123b
	sigs.k8s.io/controller-runtime v0.9.2
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/gardener/landscaper/apis => ./apis
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.1
	github.com/mandelsoft/spiff => github.com/mandelsoft/spiff v1.3.0-beta-7.0.20200909122641-3393af1d3804
	github.com/onsi/ginkgo => github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega => github.com/onsi/gomega v1.5.0
)
