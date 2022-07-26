module github.com/gardener/landscaper

go 1.17

require (
	github.com/Masterminds/sprig/v3 v3.2.2
	github.com/ahmetb/gen-crd-api-reference-docs v0.3.0
	github.com/containerd/containerd v1.5.13
	github.com/docker/cli v20.10.7+incompatible
	github.com/gardener/component-cli v0.32.0
	github.com/gardener/component-spec/bindings-go v0.0.53
	github.com/gardener/image-vector v0.6.0
	github.com/gardener/landscaper/apis v0.28.0
	github.com/gardener/landscaper/controller-utils v0.28.0
	github.com/go-logr/logr v0.4.0
	github.com/golang/mock v1.5.0
	github.com/google/uuid v1.3.0
	github.com/hashicorp/go-multierror v1.1.0
	github.com/mandelsoft/spiff v1.5.0
	github.com/mandelsoft/vfs v0.0.0-20210530103237-5249dc39ce91
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.15.0
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.2
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	github.com/xeipuuv/gojsonschema v1.2.0
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616
	golang.org/x/oauth2 v0.0.0-20210402161424-2e8d93401602
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	gopkg.in/yaml.v3 v3.0.1
	helm.sh/helm/v3 v3.7.0
	k8s.io/api v0.22.1
	k8s.io/apiextensions-apiserver v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go v0.22.1
	k8s.io/utils v0.0.0-20210802155522-efc7438f0176
	sigs.k8s.io/controller-runtime v0.10.0
	sigs.k8s.io/yaml v1.2.0
)

require (
	github.com/Microsoft/go-winio v0.5.0 // indirect
	github.com/cloudfoundry-incubator/candiedyaml v0.0.0-20170901234223-a41693b7b7af // indirect
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/klauspost/compress v1.13.6 // indirect
	golang.org/x/net v0.0.0-20211005215030-d2e5035098b3 // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20211005153810-c76a74d43a8e // indirect
	google.golang.org/grpc v1.41.0 // indirect
)

replace (
	github.com/docker/docker => github.com/moby/moby v20.10.5+incompatible
	github.com/gardener/landscaper/apis => ./apis
	github.com/gardener/landscaper/controller-utils => ./controller-utils
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.1
	github.com/mandelsoft/spiff => github.com/mandelsoft/spiff v1.3.0-beta-7.0.20200909122641-3393af1d3804
	github.com/onsi/ginkgo => github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega => github.com/onsi/gomega v1.5.0
)
