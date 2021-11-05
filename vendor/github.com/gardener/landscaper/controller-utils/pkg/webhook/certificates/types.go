// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package certificates

// ConfigInterface define functions needed for generating a specific secret.
type ConfigInterface interface {
	// GetName returns the name of the configuration.
	GetName() string
	// Generate generates a secret interface
	Generate() (DataInterface, error)
	// GenerateInfoData generates only the InfoData (metadata) which can later be used to generate a secret.
	GenerateInfoData() (InfoData, error)
	// GenerateFromInfoData combines the configuration and the provided InfoData (metadata) and generates a secret.
	GenerateFromInfoData(infoData InfoData) (DataInterface, error)
}

// DataInterface defines functions needed for defining the data map of a Kubernetes secret.
type DataInterface interface {
	// SecretData computes the data map which can be used in a Kubernetes secret.
	SecretData() map[string][]byte
}

// TypeVersion is the potentially versioned type name of an InfoData representation.
type TypeVersion string

// Unmarshaller is a factory to create a dedicated InfoData object from a byte stream
type Unmarshaller func(data []byte) (InfoData, error)

// InfoData is an interface which allows
type InfoData interface {
	TypeVersion() TypeVersion
	Marshal() ([]byte, error)
}

type emptyInfoData struct{}

func (*emptyInfoData) Marshal() ([]byte, error) {
	return nil, nil
}

func (*emptyInfoData) TypeVersion() TypeVersion {
	return ""
}

// EmptyInfoData is an infodata which does not contain any information.
var EmptyInfoData = &emptyInfoData{}
