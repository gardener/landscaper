module github.com/gardener/landscaper

go 1.13

require (
	github.com/Masterminds/sprig/v3 v3.1.0
	github.com/ahmetb/gen-crd-api-reference-docs v0.2.0
	github.com/alecthomas/jsonschema v0.0.0-20200530073317-71f438968921
	github.com/cloudfoundry-incubator/candiedyaml v0.0.0-20170901234223-a41693b7b7af // indirect
	github.com/containerd/containerd v1.4.2
	github.com/docker/cli v20.10.0-rc1+incompatible
	github.com/gardener/component-cli v0.4.1-0.20210115155235-6f6c40d7d28a
	github.com/gardener/component-spec/bindings-go v0.0.28-0.20210119120517-8a3773c214e9
	github.com/go-logr/logr v0.3.0
	github.com/go-logr/zapr v0.3.0
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/google/uuid v1.1.1
	github.com/hashicorp/go-multierror v1.1.0
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/imdario/mergo v0.3.9 // indirect
	github.com/mandelsoft/spiff v1.5.0
	github.com/mandelsoft/vfs v0.0.0-20201002134249-3c471f64a4d1
	github.com/onsi/ginkgo v1.14.0
	github.com/onsi/gomega v1.10.1
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.1
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415
	github.com/xeipuuv/gojsonschema v1.2.0
	go.uber.org/zap v1.16.0
	golang.org/x/lint v0.0.0-20201208152925-83fdc39ff7b5
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	golang.org/x/time v0.0.0-20200416051211-89c76fbcd5d1 // indirect
	gomodules.xyz/jsonpatch/v2 v2.1.0 // indirect
	google.golang.org/appengine v1.6.6 // indirect
	google.golang.org/genproto v0.0.0-20201019141844-1ed22bb0c154 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
	helm.sh/helm/v3 v3.2.4
	k8s.io/api v0.19.5
	k8s.io/apimachinery v0.19.5
	k8s.io/client-go v0.19.5
	k8s.io/code-generator v0.19.5
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920
	sigs.k8s.io/controller-runtime v0.6.0
	sigs.k8s.io/controller-tools v0.3.1-0.20200517180335-820a4a27ea84 // including a fix from master
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.1
	github.com/mandelsoft/spiff => github.com/mandelsoft/spiff v1.3.0-beta-7.0.20200909122641-3393af1d3804
	github.com/onsi/ginkgo => github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega => github.com/onsi/gomega v1.5.0
)
