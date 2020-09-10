module github.com/gardener/landscaper

go 1.13

require (
	github.com/Masterminds/sprig/v3 v3.1.0
	github.com/containerd/containerd v1.3.2
	github.com/deislabs/oras v0.8.1
	github.com/docker/cli v0.0.0-20200130152716-5d0cf8839492
	github.com/gardener/component-spec/bindings-go v0.0.0-00010101000000-000000000000
	github.com/go-logr/logr v0.1.0
	github.com/go-logr/zapr v0.1.1
	github.com/go-openapi/spec v0.19.3
	github.com/go-openapi/strfmt v0.19.3
	github.com/go-openapi/validate v0.19.5
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/mock v1.2.0
	github.com/golang/protobuf v1.4.0 // indirect
	github.com/google/uuid v1.1.1
	github.com/hashicorp/go-multierror v1.1.0
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/imdario/mergo v0.3.9 // indirect
	github.com/kr/pretty v0.2.0 // indirect
	github.com/mandelsoft/vfs v0.0.0-20200909114934-2adb28762d89
	github.com/onsi/ginkgo v1.14.0
	github.com/onsi/gomega v1.10.1
	github.com/opencontainers/go-digest v1.0.0-rc1
	github.com/opencontainers/image-spec v1.0.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.5.1 // indirect
	github.com/prometheus/procfs v0.0.11 // indirect
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	go.uber.org/zap v1.15.0
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	golang.org/x/time v0.0.0-20200416051211-89c76fbcd5d1 // indirect
	gomodules.xyz/jsonpatch/v2 v2.1.0 // indirect
	google.golang.org/appengine v1.6.6 // indirect
	helm.sh/helm/v3 v3.2.4
	k8s.io/api v0.18.2
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v0.18.2
	k8s.io/code-generator v0.18.2
	k8s.io/gengo v0.0.0-20200413195148-3a45101e95ac // indirect
	k8s.io/utils v0.0.0-20200414100711-2df71ebbae66
	rsc.io/letsencrypt v0.0.3 // indirect
	sigs.k8s.io/controller-runtime v0.6.0
	sigs.k8s.io/controller-tools v0.3.1-0.20200517180335-820a4a27ea84 // including a fix from master
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/gardener/component-spec/bindings-go => github.com/schrodit/component-spec/bindings-go v0.0.0-20200814142028-161f0f56a2f2
	github.com/onsi/ginkgo => github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega => github.com/onsi/gomega v1.5.0
)
