module github.com/gardener/landscaper/apis

go 1.16

require (
	github.com/gardener/component-spec/bindings-go v0.0.41
	github.com/go-openapi/jsonreference v0.19.3
	github.com/go-openapi/spec v0.19.3
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.4
	k8s.io/api v0.19.11
	k8s.io/apimachinery v0.19.11
	k8s.io/code-generator v0.18.2
	k8s.io/kube-openapi v0.0.0-20201113171705-d219536bb9fd
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009
)

replace (
	k8s.io/api => k8s.io/api v0.19.11
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.11
	// need to use k8s v19 until a bug in 1.20 is fixed: https://github.com/kubernetes/kubernetes/issues/98380
	k8s.io/code-generator => k8s.io/code-generator v0.19.11
)
