// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/gardener/landscaper/pkg/utils/webhook/certificates"
)

const (
	certSecretName = "landscaper-webhook-cert"
)

// The DNSNamesConfig holds the list of DNS names used in the certificates.CertificateSecretConfig.
type DNSNamesConfig interface {
	Get() []string
}

// The DNSNamespacedNameConfig derives the DNS names from the service name and service namespace.
type DNSNamespacedNameConfig struct {
	config []string
}

// The DNSWebhookURLConfig derives the DNS names directly from the webhook url.
type DNSWebhookURLConfig struct {
	config []string
}

func (c *DNSNamespacedNameConfig) Set(name, namespace string) {
	c.config = []string{
		name,
		fmt.Sprintf("%s.%s", name, namespace),
		fmt.Sprintf("%s.%s.svc", name, namespace),
	}
}

func (c *DNSNamespacedNameConfig) Get() []string {
	return c.config
}

func (c *DNSWebhookURLConfig) Set(webhookURL string) error {
	parsedURL, err := url.Parse(webhookURL)
	if err != nil {
		return err
	}
	dnsName, _, err := net.SplitHostPort(parsedURL.Host)
	if err != nil {
		// If SplitHostPort fails here, it is due to a missing port.
		// In this case the host of the parsed URL is used directly.
		c.config = []string{
			parsedURL.Host,
		}
	} else {
		c.config = []string{
			dnsName,
		}
	}
	return nil
}

func (c *DNSWebhookURLConfig) Get() []string {
	return c.config
}

// GenerateCertificates generates the certificates that are required for a webhook. It returns the ca bundle, and it
// stores the server certificate and key locally on the file system.
func GenerateCertificates(ctx context.Context, kubeClient client.Client, certDir, namespace string, dnsNamesConfig DNSNamesConfig) ([]byte, error) {
	var (
		caCert     *certificates.Certificate
		serverCert *certificates.Certificate
		err        error
	)

	caConfig := &certificates.CertificateSecretConfig{
		CommonName: "webhook-ca",
		CertType:   certificates.CACert,
		PKCS:       certificates.PKCS8,
	}

	// If the namespace is not set then the webhook controller is running locally. We simply generate a new certificate in this case.
	if len(namespace) == 0 {
		caCert, serverCert, err = GenerateNewCAAndServerCert(dnsNamesConfig, *caConfig)
		if err != nil {
			return nil, errors.Wrapf(err, "error generating new certificates for webhook server")
		}
		return writeCertificates(certDir, caCert, serverCert)
	}

	// The controller stores the generated webhook certificate in a secret in the cluster. It tries to read it. If it does not exist a
	// new certificate is generated.

	generateAndUpdateCertificate := func() ([]byte, error) {
		// The secret was not found, let's generate new certificates and store them in the secret afterwards.
		caCert, serverCert, err = GenerateNewCAAndServerCert(dnsNamesConfig, *caConfig)
		if err != nil {
			return nil, errors.Wrapf(err, "error generating new certificates for webhook server")
		}

		secret := &corev1.Secret{}
		secret.ObjectMeta = metav1.ObjectMeta{Namespace: namespace, Name: certSecretName}
		secret.Type = corev1.SecretTypeOpaque
		secret.Data = map[string][]byte{
			certificates.DataKeyCertificateCA: caCert.CertificatePEM,
			certificates.DataKeyPrivateKeyCA:  caCert.PrivateKeyPEM,
			certificates.DataKeyCertificate:   serverCert.CertificatePEM,
			certificates.DataKeyPrivateKey:    serverCert.PrivateKeyPEM,
		}

		if _, err := controllerutil.CreateOrUpdate(ctx, kubeClient, secret, func() error {
			secret.Type = corev1.SecretTypeOpaque
			secret.Data = map[string][]byte{
				certificates.DataKeyCertificateCA: caCert.CertificatePEM,
				certificates.DataKeyPrivateKeyCA:  caCert.PrivateKeyPEM,
				certificates.DataKeyCertificate:   serverCert.CertificatePEM,
				certificates.DataKeyPrivateKey:    serverCert.PrivateKeyPEM,
			}
			return nil
		}); err != nil {
			return nil, err
		}

		return writeCertificates(certDir, caCert, serverCert)
	}

	secret := &corev1.Secret{}
	if err := kubeClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: certSecretName}, secret); err != nil {
		return generateAndUpdateCertificate()
	}

	// The secret has been found and we are now trying to read the stored certificate inside it.
	caCert, serverCert, err = loadExistingCAAndServerCert(secret.Data, caConfig.PKCS)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading data of secret %s/%s", namespace, certSecretName)
	}

	// update certificates if the dns names have changed
	if !sets.NewString(caCert.Certificate.DNSNames...).HasAll(dnsNamesConfig.Get()...) {
		return generateAndUpdateCertificate()
	}

	return writeCertificates(certDir, caCert, serverCert)
}

// GenerateNewCAAndServerCert generates a new ca and server certificate for a service in a name and namespace.
func GenerateNewCAAndServerCert(dnsNamesConfig DNSNamesConfig, caConfig certificates.CertificateSecretConfig) (*certificates.Certificate, *certificates.Certificate, error) {
	caCert, err := caConfig.GenerateCertificate()
	if err != nil {
		return nil, nil, err
	}

	var (
		ipAddresses []net.IP
	)

	serverConfig := &certificates.CertificateSecretConfig{
		CommonName:  "landscaper-webhooks",
		DNSNames:    dnsNamesConfig.Get(),
		IPAddresses: ipAddresses,
		CertType:    certificates.ServerCert,
		SigningCA:   caCert,
		PKCS:        caConfig.PKCS,
	}

	serverCert, err := serverConfig.GenerateCertificate()
	if err != nil {
		return nil, nil, err
	}

	return caCert, serverCert, nil
}

func loadExistingCAAndServerCert(data map[string][]byte, pkcs int) (*certificates.Certificate, *certificates.Certificate, error) {
	secretDataCACert, ok := data[certificates.DataKeyCertificateCA]
	if !ok {
		return nil, nil, fmt.Errorf("secret does not contain %s key", certificates.DataKeyCertificateCA)
	}
	secretDataCAKey, ok := data[certificates.DataKeyPrivateKeyCA]
	if !ok {
		return nil, nil, fmt.Errorf("secret does not contain %s key", certificates.DataKeyPrivateKeyCA)
	}
	caCert, err := certificates.LoadCertificate("", secretDataCAKey, secretDataCACert, pkcs)
	if err != nil {
		return nil, nil, fmt.Errorf("could not load ca certificate")
	}

	secretDataServerCert, ok := data[certificates.DataKeyCertificate]
	if !ok {
		return nil, nil, fmt.Errorf("secret does not contain %s key", certificates.DataKeyCertificate)
	}
	secretDataServerKey, ok := data[certificates.DataKeyPrivateKey]
	if !ok {
		return nil, nil, fmt.Errorf("secret does not contain %s key", certificates.DataKeyPrivateKey)
	}
	serverCert, err := certificates.LoadCertificate("", secretDataServerKey, secretDataServerCert, pkcs)
	if err != nil {
		return nil, nil, fmt.Errorf("could not load server certificate")
	}

	return caCert, serverCert, nil
}

func writeCertificates(certDir string, caCert, serverCert *certificates.Certificate) ([]byte, error) {
	var (
		serverKeyPath  = filepath.Join(certDir, certificates.DataKeyPrivateKey)
		serverCertPath = filepath.Join(certDir, certificates.DataKeyCertificate)
	)

	if err := os.MkdirAll(certDir, 0755); err != nil {
		return nil, err
	}
	if err := ioutil.WriteFile(serverKeyPath, serverCert.PrivateKeyPEM, 0666); err != nil {
		return nil, err
	}
	if err := ioutil.WriteFile(serverCertPath, serverCert.CertificatePEM, 0666); err != nil {
		return nil, err
	}

	return caCert.CertificatePEM, nil
}

func StringArrayIncludes(list []string, expects ...string) bool {
	actual := sets.NewString(list...)
	return actual.HasAll(expects...)
}
