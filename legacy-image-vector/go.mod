module github.com/gardener/landscaper/legacy-image-vector

go 1.25.4

require (
	github.com/Masterminds/semver/v3 v3.4.0
	github.com/gardener/landscaper/legacy-component-spec/bindings-go v0.0.0-00010101000000-000000000000
	github.com/go-logr/logr v1.4.3
	github.com/onsi/ginkgo v1.14.0
	github.com/onsi/ginkgo/v2 v2.27.2
	github.com/onsi/gomega v1.38.2
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.1.1
	sigs.k8s.io/yaml v1.6.0
)

replace github.com/gardener/landscaper/legacy-component-spec/bindings-go => ../legacy-component-spec/bindings-go

require (
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mandelsoft/filepath v0.0.0-20240223090642-3e2777258aa3 // indirect
	github.com/mandelsoft/vfs v0.4.4 // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/nxadm/tail v1.4.11 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/net v0.46.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/apimachinery v0.34.2 // indirect
)
