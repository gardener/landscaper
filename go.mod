module github.com/gardener/landscaper

go 1.13

require (
	github.com/PaesslerAG/jsonpath v0.1.1 // indirect
	github.com/Pallinder/go-randomdata v1.2.0 // indirect
	github.com/cstockton/go-conv v0.0.0-20170524002450-66a2b2ba36e1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.1.0
	github.com/go-logr/zapr v0.1.1
	github.com/go-openapi/spec v0.19.3
	github.com/go-openapi/strfmt v0.19.3
	github.com/go-openapi/stubs v0.0.0-20170429194734-98bff229fc7b // indirect
	github.com/go-openapi/validate v0.19.5
	github.com/gobuffalo/flect v0.2.0 // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/mock v1.2.0
	github.com/golang/protobuf v1.4.0 // indirect
	github.com/hashicorp/go-multierror v1.1.0
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/imdario/mergo v0.3.9 // indirect
	github.com/kr/pretty v0.2.0 // indirect
	github.com/manveru/faker v0.0.0-20171103152722-9fbc68a78c4d // indirect
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.8.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.5.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/procfs v0.0.11 // indirect
	github.com/spf13/afero v1.2.2
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/zach-klippenstein/goregen v0.0.0-20160303162051-795b5e3961ea // indirect
	go.uber.org/zap v1.15.0
	golang.org/x/crypto v0.0.0-20200427165652-729f1e841bcc // indirect
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b
	golang.org/x/net v0.0.0-20200425230154-ff2c4b7c35a0 // indirect
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	golang.org/x/sys v0.0.0-20200427175716-29b57079015a // indirect
	golang.org/x/time v0.0.0-20200416051211-89c76fbcd5d1 // indirect
	gomodules.xyz/jsonpatch/v2 v2.1.0 // indirect
	google.golang.org/appengine v1.6.6 // indirect
	gopkg.in/yaml.v3 v3.0.0-20190905181640-827449938966 // indirect
	k8s.io/api v0.18.2
	k8s.io/apiextensions-apiserver v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v0.18.2
	k8s.io/code-generator v0.18.2
	k8s.io/gengo v0.0.0-20200413195148-3a45101e95ac // indirect
	k8s.io/kube-openapi v0.0.0-20200121204235-bf4fb3bd569c
	k8s.io/utils v0.0.0-20200414100711-2df71ebbae66
	sigs.k8s.io/controller-runtime v0.6.0
	sigs.k8s.io/controller-tools v0.3.1-0.20200517180335-820a4a27ea84 // including a fix from master
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/onsi/ginkgo => github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega => github.com/onsi/gomega v1.5.0
)
