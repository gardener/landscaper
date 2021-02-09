module github.com/gardener/landscaper/apis

go 1.13

require (
	github.com/gardener/component-spec/bindings-go v0.0.27
	github.com/go-logr/logr v0.3.0 // indirect
	github.com/mandelsoft/vfs v0.0.0-20201002134249-3c471f64a4d1
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.4
	golang.org/x/tools v0.0.0-20201002184944-ecd9fd270d5d // indirect
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/code-generator v0.20.2
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009
)

replace (
	k8s.io/api => k8s.io/api v0.19.7
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.7
	k8s.io/code-generator => k8s.io/code-generator v0.19.7
)
