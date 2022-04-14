module github.com/gardener/landscaper/controller-utils

go 1.16

require (
	github.com/gardener/landscaper/apis v0.23.0
	github.com/go-logr/logr v0.4.0
	github.com/go-logr/zapr v0.4.0
	github.com/golang/mock v1.4.1
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.15.0
	github.com/pkg/errors v0.9.1
	github.com/spf13/pflag v1.0.5
	go.uber.org/zap v1.19.0
	golang.org/x/crypto v0.0.0-20210220033148-5ea612d1eb83
	k8s.io/api v0.22.1
	k8s.io/apiextensions-apiserver v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go v0.22.1
	sigs.k8s.io/controller-runtime v0.10.0
	sigs.k8s.io/yaml v1.2.0
)

replace github.com/gardener/landscaper/apis => ../apis
