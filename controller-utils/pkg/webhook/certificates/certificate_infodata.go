// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package certificates

import (
	"encoding/json"
	"fmt"
)

// CertificateDataType is the type used to denote an CertificateJSONData structure in the ShootState
const CertificateDataType = TypeVersion("certificate")

// CertificateJSONData is the json representation of CertificateInfoData used to store Certificate metadata in the ShootState
type CertificateJSONData struct {
	PrivateKey  []byte `json:"privateKey"`
	Certificate []byte `json:"certificate"`
}

// UnmarshalCert unmarshals an CertificateJSONData into a CertificateInfoData.
func UnmarshalCert(bytes []byte) (InfoData, error) {
	if bytes == nil {
		return nil, fmt.Errorf("no data given")
	}
	data := &CertificateJSONData{}
	err := json.Unmarshal(bytes, data)
	if err != nil {
		return nil, err
	}

	return NewCertificateInfoData(data.PrivateKey, data.Certificate), nil
}

// CertificateInfoData holds a certificate's private key data and certificate data.
type CertificateInfoData struct {
	PrivateKey  []byte
	Certificate []byte
}

// TypeVersion implements InfoData
func (c *CertificateInfoData) TypeVersion() TypeVersion {
	return CertificateDataType
}

// Marshal implements InfoData
func (c *CertificateInfoData) Marshal() ([]byte, error) {
	return json.Marshal(&CertificateJSONData{c.PrivateKey, c.Certificate})
}

// NewCertificateInfoData creates a new CertificateInfoData struct
func NewCertificateInfoData(privateKey, certificate []byte) *CertificateInfoData {
	return &CertificateInfoData{privateKey, certificate}
}

var types = map[TypeVersion]Unmarshaller{
	CertificateDataType: UnmarshalCert,
	PrivateKeyDataType:  UnmarshalPrivateKey,
}

// GetUnmarshaller returns an Unmarshaller for the given typeName.
func GetUnmarshaller(typeName TypeVersion) Unmarshaller {
	return types[typeName]
}
