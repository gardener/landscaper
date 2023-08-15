// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"

	utilerrors "github.com/gardener/landscaper/controller-utils/pkg/errors"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/controller-utils/pkg/webhook/certificates"
)

// GeDNSNamesFromNamespacedName creates a list of DNS names derived from a service name and namespace.
func GeDNSNamesFromNamespacedName(namespace, name string) []string {
	return []string{
		name,
		fmt.Sprintf("%s.%s", name, namespace),
		fmt.Sprintf("%s.%s.svc", name, namespace),
	}
}

// GetDNSNamesFromURL creates a list of DNS names derived from a URL.
func GetDNSNamesFromURL(rawurl string) ([]string, error) {
	parsedURL, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	dnsName, _, err := net.SplitHostPort(parsedURL.Host)
	if err != nil {
		// If SplitHostPort fails here, it is due to a missing port.
		// In this case the host of the parsed URL is used directly.
		return []string{
			parsedURL.Host,
		}, nil
	} else {
		return []string{
			dnsName,
		}, nil
	}
}

// GenerateCertificates generates the certificates that are required for a webhook. It returns the generated certificates,
// and it stores the server certificate and key locally on the file system.
// The first return value is the CA certificate, the second one the server certificate.
func GenerateCertificates(ctx context.Context, kubeClient client.Client, certDir, namespace, name, certSecretName string,
	dnsNames []string) (*certificates.Certificate, *certificates.Certificate, error) {

	logger, ctx := logging.FromContextOrNew(ctx, nil, lc.KeyMethod, "GenerateCertificates")

	caConfig := &certificates.CertificateSecretConfig{
		CommonName: "webhook-ca",
		CertType:   certificates.CACert,
		PKCS:       certificates.PKCS8,
	}

	// The controller stores the generated webhook certificate in a secret in the cluster. It tries to read it. If it does not exist a
	// new certificate is generated.

	logger.Info("GenerateCertificates: fetch secrets")

	secret := &corev1.Secret{}
	if err := kubeClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: certSecretName}, secret); err != nil {
		logger.Info("GenerateCertificates: fetch secrets failed")

		if !apierrors.IsNotFound(err) {
			logger.Info("GenerateCertificates: fetch failed hard")
			return nil, nil, errors.Wrapf(err, "error fetching secret for webhook server")
		}

		logger.Info("GenerateCertificates: generate new cert")
		caCert, serverCert, err := generateNewCAAndServerCert(name, dnsNames, *caConfig)
		if err != nil {
			logger.Info("GenerateCertificates: generate new cert failed")
			return nil, nil, errors.Wrapf(err, "error generating new certificates for webhook server")
		}

		err = createOrUpdateSecret(ctx, kubeClient, caCert, serverCert, namespace, certSecretName, true)
		if err == nil {
			logger.Info("GenerateCertificates: new secret generated")
			return caCert, serverCert, writeCertificate(certDir, serverCert)
		} else {
			errs := utilerrors.NewErrorList(err)
			// try to refetch secret if it was created by another replica
			logger.Info("GenerateCertificates: new secret generation failed")
			if err := kubeClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: certSecretName}, secret); err != nil {
				if apierrors.IsNotFound(err) {
					return nil, nil, errors.Wrapf(errs.Aggregate(), "error writing certificates into secret")
				}
				errs.Append(err)
				logger.Info("GenerateCertificates: fetching new secret failed")
				return nil, nil, errors.Wrapf(errs.Aggregate(), "unable to write certificates into secret and check for existing secret")
			}
		}
	}

	// The secret has been found and we are now trying to read the stored certificate inside it and updates it if
	// required
	logger.Info("GenerateCertificates: loadAndUpdateSecret start")
	caCert, serverCert, retry, err := loadAndUpdateSecret(ctx, kubeClient, secret, name, dnsNames, caConfig)
	if err != nil {

		if retry {
			logger.Info("GenerateCertificates: loadAndUpdateSecret failed and retried")
			secret = &corev1.Secret{}
			if err := kubeClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: certSecretName}, secret); err != nil {
				logger.Info("GenerateCertificates: fetch updated secret failed")
				return nil, nil, err
			}
			caCert, serverCert, _, err = loadAndUpdateSecret(ctx, kubeClient, secret, name, dnsNames, caConfig)
			if err != nil {
				logger.Info("GenerateCertificates: loadAndUpdateSecret retry failed")
				return nil, nil, err
			}
		} else {
			logger.Info("GenerateCertificates: loadAndUpdateSecret failed")
			return nil, nil, err
		}

	}

	logger.Info("GenerateCertificates: cert returned")
	return caCert, serverCert, writeCertificate(certDir, serverCert)
}

func loadAndUpdateSecret(ctx context.Context, kubeClient client.Client, secret *corev1.Secret, name string, dnsNames []string,
	caConfig *certificates.CertificateSecretConfig) (*certificates.Certificate, *certificates.Certificate, bool, error) {

	logger, ctx := logging.FromContextOrNew(ctx, nil)

	logger.Info("loadAndUpdateSecret: load existing cert")

	caCert, serverCert, err := loadExistingCAAndServerCert(secret.Data, caConfig.PKCS)
	if err != nil {
		logger.Info("loadAndUpdateSecret: load existing cert failed")
		return nil, nil, false, errors.Wrapf(err, "error reading data of secret %s/%s", secret.Namespace, secret.Name)
	}

	// update certificates if the dns names have changed
	if !sets.NewString(serverCert.Certificate.DNSNames...).HasAll(dnsNames...) {
		logger.Info("loadAndUpdateSecret: dns names have changed")

		caCert, serverCert, err = generateNewCAAndServerCert(name, dnsNames, *caConfig)
		if err != nil {
			logger.Info("loadAndUpdateSecret: generate new cert failed")
			return nil, nil, false, errors.Wrapf(err, "error generating new certificates for webhook server")
		}

		if err = createOrUpdateSecret(ctx, kubeClient, caCert, serverCert, secret.Namespace, secret.Name, false); err != nil {
			logger.Info("loadAndUpdateSecret: updating secret failed")
			return nil, nil, true, errors.Wrapf(err, "error updating secret for webhook")
		}
	}

	logger.Info("loadAndUpdateSecret: return cert data")
	return caCert, serverCert, false, nil
}

func createOrUpdateSecret(ctx context.Context, kubeClient client.Client, caCert, serverCert *certificates.Certificate,
	namespace, certSecretName string, create bool) error {
	// The secret was not found, let's generate new certificates and store them in the secret afterwards.
	secret := &corev1.Secret{}
	secret.ObjectMeta = metav1.ObjectMeta{Namespace: namespace, Name: certSecretName}
	secret.Type = corev1.SecretTypeOpaque
	secret.Data = map[string][]byte{
		certificates.DataKeyCertificateCA: caCert.CertificatePEM,
		certificates.DataKeyPrivateKeyCA:  caCert.PrivateKeyPEM,
		certificates.DataKeyCertificate:   serverCert.CertificatePEM,
		certificates.DataKeyPrivateKey:    serverCert.PrivateKeyPEM,
	}

	if create {
		return kubeClient.Create(ctx, secret)
	} else {
		return kubeClient.Update(ctx, secret)
	}
}

// GenerateNewCAAndServerCert generates a new ca and server certificate for a service in a name and namespace.
func generateNewCAAndServerCert(name string, dnsNames []string, caConfig certificates.CertificateSecretConfig) (*certificates.Certificate, *certificates.Certificate, error) {
	caCert, err := caConfig.GenerateCertificate()
	if err != nil {
		return nil, nil, err
	}

	var (
		ipAddresses []net.IP
	)

	serverConfig := &certificates.CertificateSecretConfig{
		CommonName:  name,
		DNSNames:    dnsNames,
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

// writeCertificate writes the given certificates to the local filesystem.
func writeCertificate(certDir string, cert *certificates.Certificate) error {
	var (
		serverKeyPath  = filepath.Join(certDir, certificates.DataKeyPrivateKey)
		serverCertPath = filepath.Join(certDir, certificates.DataKeyCertificate)
	)

	if err := os.MkdirAll(certDir, 0755); err != nil {
		return nil
	}
	if err := os.WriteFile(serverKeyPath, cert.PrivateKeyPEM, 0666); err != nil {
		return nil
	}
	if err := os.WriteFile(serverCertPath, cert.CertificatePEM, 0666); err != nil {
		return nil
	}

	return nil
}

// readCertificate is the sibling function to writeCertificates and reads a certificate including its CA from the local filesystem.
// Uses PKCS8 by default.
// name is the name which will be given to the certificate, not the filename.
// func readCertificate(certDir string, name string) (*certificates.Certificate, error) {
// 	var (
// 		keyPath  = filepath.Join(certDir, certificates.DataKeyPrivateKey)
// 		certPath = filepath.Join(certDir, certificates.DataKeyCertificate)
// 	)

// 	keyBytes, err := os.ReadFile(keyPath)
// 	if err != nil {
// 		return nil, err
// 	}
// 	certBytes, err := os.ReadFile(certPath)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return certificates.LoadCertificate(name, keyBytes, certBytes, certificates.PKCS8)
// }

func StringArrayIncludes(list []string, expects ...string) bool {
	actual := sets.NewString(list...)
	return actual.HasAll(expects...)
}
