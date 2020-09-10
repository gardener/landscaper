package x509

import (
	"github.com/mandelsoft/spiff/dynaml"
)

func init() {
	dynaml.RegisterValidator("publickey", ValPublicKey)
	dynaml.RegisterValidator("privatekey", ValPrivateKey)
	dynaml.RegisterValidator("certificate", ValCertificate)
	dynaml.RegisterValidator("ca", ValCA)
}
func ValPrivateKey(value interface{}, binding dynaml.Binding, args ...interface{}) (bool, string, error, bool) {
	s, err := dynaml.StringValue("privatekey", value)
	if err != nil {
		return dynaml.ValidatorErrorf("%s", err)
	}
	_, err = ParsePrivateKey(s)
	return dynaml.SimpleValidatorResult(err == nil, "is private key", "is no private key: %s", err)
}

func ValCertificate(value interface{}, binding dynaml.Binding, args ...interface{}) (bool, string, error, bool) {
	s, err := dynaml.StringValue("certificate", value)
	if err != nil {
		return dynaml.ValidatorErrorf("%s", err)
	}
	_, err = ParseCertificate(s)
	return dynaml.SimpleValidatorResult(err == nil, "is certificate", "is no certificate: %s", err)
}

func ValCA(value interface{}, binding dynaml.Binding, args ...interface{}) (bool, string, error, bool) {
	s, err := dynaml.StringValue("ca", value)
	if err != nil {
		return dynaml.ValidatorErrorf("%s", err)
	}
	c, err := ParseCertificate(s)
	if err != nil {
		return dynaml.ValidatorResult(false, "is no certificate: %s", err)
	}
	if !c.IsCA {
		return dynaml.ValidatorResult(false, "is no ca certificate: %s", err)
	}
	return dynaml.ValidatorResult(true, "is ca")
}

func ValPublicKey(value interface{}, binding dynaml.Binding, args ...interface{}) (bool, string, error, bool) {
	s, err := dynaml.StringValue("publickey", value)
	if err != nil {
		return dynaml.ValidatorErrorf("%s", err)
	}
	_, err = ParsePublicKey(s)
	return dynaml.SimpleValidatorResult(err == nil, "is public key", "is no public key: %s", err)
}
