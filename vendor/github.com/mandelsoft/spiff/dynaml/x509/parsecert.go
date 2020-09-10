package x509

import (
	. "github.com/mandelsoft/spiff/dynaml"
	"github.com/mandelsoft/spiff/yaml"
	"net"
	"time"
)

const F_ParseCert = "x509parsecert"

func init() {
	RegisterFunction(F_ParseCert, func_x509parsecert)
}

func func_x509parsecert(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	var err error
	info := DefaultInfo()

	if len(arguments) != 1 {
		return info.Error("invalid argument count for %s", F_ParseCert)
	}

	pemstr, ok := arguments[0].(string)
	if !ok {
		return info.Error("argument for %s must be a certificate in pem format", F_ParseCert)
	}

	cert, err := ParseCertificate(pemstr)
	if err != nil {
		return info.Error("argument for %s must be a certificate in pem format: %s", F_ParseCert, err)
	}

	result := map[string]yaml.Node{}

	result["isCA"] = NewNode(cert.IsCA, binding)
	if cert.Subject.CommonName != "" {
		result["commonName"] = NewNode(cert.Subject.CommonName, binding)
	}
	if cert.Subject.Country != nil {
		result["country"] = NodeStringList(cert.Subject.Country, binding)
	}
	if cert.Subject.Organization != nil {
		result["organization"] = NodeStringList(cert.Subject.Organization, binding)
	}

	if cert.DNSNames != nil {
		result["dnsNames"] = NodeStringList(cert.DNSNames, binding)
	}
	if cert.IPAddresses != nil {
		result["ipAddresses"] = NodeIPList(cert.IPAddresses, binding)
	}
	if len(cert.DNSNames)+len(cert.DNSNames) > 0 {
		result["hosts"] = NewNode(
			append(NodeIPList(cert.IPAddresses, binding).Value().([]yaml.Node),
				NodeStringList(cert.DNSNames, binding).Value().([]yaml.Node)...), binding)
	}

	result["validFrom"] = NewNode(cert.NotBefore.Format("Jan 2 15:04:05 2006"), binding)
	result["validUntil"] = NewNode(cert.NotAfter.Format("Jan 2 15:04:05 2006"), binding)
	result["validity"] = NewNode(int64(cert.NotAfter.Sub(time.Now())/time.Hour), binding)

	result["usage"] = NodeStringList(append(KeyUsages(cert.KeyUsage), ExtKeyUsages(cert.ExtKeyUsage)...), binding)

	if cert.PublicKey != nil {
		str, err := PublicKeyPEM(publicKey(cert))
		if err != nil {
			return info.Error("%s", err)
		}
		result["publicKey"] = NewNode(str, binding)
	}

	return result, info, true
}

func NodeStringList(list []string, binding Binding) yaml.Node {
	nodelist := make([]yaml.Node, len(list))

	for i, s := range list {
		nodelist[i] = NewNode(s, binding)
	}
	return NewNode(nodelist, binding)
}

func NodeIPList(list []net.IP, binding Binding) yaml.Node {
	nodelist := make([]yaml.Node, len(list))

	for i, s := range list {
		nodelist[i] = NewNode(s.String(), binding)
	}
	return NewNode(nodelist, binding)
}
