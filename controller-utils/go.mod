module github.com/gardener/landscaper/controller-utils

go 1.16

require (
	github.com/gardener/landscaper/apis v0.0.0-00010101000000-000000000000
	github.com/go-logr/logr v0.4.0
	github.com/pkg/errors v0.9.1
	golang.org/x/crypto v0.0.0-20210220033148-5ea612d1eb83
	k8s.io/api v0.22.1
	k8s.io/apiextensions-apiserver v0.22.1
	k8s.io/apimachinery v0.22.1
	sigs.k8s.io/controller-runtime v0.10.0
)

replace github.com/gardener/landscaper/apis => ../apis
