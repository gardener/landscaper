// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/landscaper/pkg/utils/webhook/certificates"
)

const (
	certSecretName = "landscaper-webhook-cert"
)

// GenerateCertificates generates the certificates that are required for a webhook. It returns the ca bundle, and it
// stores the server certificate and key locally on the file system.
func GenerateCertificates(ctx context.Context, mgr manager.Manager, certDir, namespace, name string) ([]byte, error) {
	var (
		caCert     *certificates.Certificate
		serverCert *certificates.Certificate
		err        error
	)

	// If the namespace is not set then the webhook controller is running locally. We simply generate a new certificate in this case.
	if len(namespace) == 0 {
		caCert, serverCert, err = generateNewCAAndServerCert(namespace, name)
		if err != nil {
			return nil, errors.Wrapf(err, "error generating new certificates for webhook server")
		}
		return writeCertificates(certDir, caCert, serverCert)
	}

	// The controller stores the generated webhook certificate in a secret in the cluster. It tries to read it. If it does not exist a
	// new certificate is generated.
	c, err := getClient(mgr)
	if err != nil {
		return nil, err
	}

	secret := &corev1.Secret{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: certSecretName}, secret); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, errors.Wrapf(err, "error getting cert secret")
		}

		// The secret was not found, let's generate new certificates and store them in the secret afterwards.
		caCert, serverCert, err = generateNewCAAndServerCert(namespace, name)
		if err != nil {
			return nil, errors.Wrapf(err, "error generating new certificates for webhook server")
		}

		secret.ObjectMeta = metav1.ObjectMeta{Namespace: namespace, Name: certSecretName}
		secret.Type = corev1.SecretTypeOpaque
		secret.Data = map[string][]byte{
			certificates.DataKeyCertificateCA: caCert.CertificatePEM,
			certificates.DataKeyPrivateKeyCA:  caCert.PrivateKeyPEM,
			certificates.DataKeyCertificate:   serverCert.CertificatePEM,
			certificates.DataKeyPrivateKey:    serverCert.PrivateKeyPEM,
		}
		if err := c.Create(ctx, secret); err != nil {
			return nil, err
		}

		return writeCertificates(certDir, caCert, serverCert)
	}

	// The secret has been found and we are now trying to read the stored certificate inside it.
	caCert, serverCert, err = loadExistingCAAndServerCert(secret.Data)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading data of secret %s/%s", namespace, certSecretName)
	}
	return writeCertificates(certDir, caCert, serverCert)
}

func generateNewCAAndServerCert(namespace, name string) (*certificates.Certificate, *certificates.Certificate, error) {
	caConfig := &certificates.CertificateSecretConfig{
		CommonName: "webhook-ca",
		CertType:   certificates.CACert,
	}

	caCert, err := caConfig.GenerateCertificate()
	if err != nil {
		return nil, nil, err
	}

	var (
		dnsNames    []string
		ipAddresses []net.IP
	)

	dnsNames = []string{
		name,
		fmt.Sprintf("%s.%s", name, namespace),
		fmt.Sprintf("%s.%s.svc", name, namespace),
	}

	serverConfig := &certificates.CertificateSecretConfig{
		CommonName:  name,
		DNSNames:    dnsNames,
		IPAddresses: ipAddresses,
		CertType:    certificates.ServerCert,
		SigningCA:   caCert,
	}

	serverCert, err := serverConfig.GenerateCertificate()
	if err != nil {
		return nil, nil, err
	}

	return caCert, serverCert, nil
}

func loadExistingCAAndServerCert(data map[string][]byte) (*certificates.Certificate, *certificates.Certificate, error) {
	secretDataCACert, ok := data[certificates.DataKeyCertificateCA]
	if !ok {
		return nil, nil, fmt.Errorf("secret does not contain %s key", certificates.DataKeyCertificateCA)
	}
	secretDataCAKey, ok := data[certificates.DataKeyPrivateKeyCA]
	if !ok {
		return nil, nil, fmt.Errorf("secret does not contain %s key", certificates.DataKeyPrivateKeyCA)
	}
	caCert, err := certificates.LoadCertificate("", secretDataCAKey, secretDataCACert)
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
	serverCert, err := certificates.LoadCertificate("", secretDataServerKey, secretDataServerCert)
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

func getClient(mgr manager.Manager) (client.Client, error) {
	s := runtime.NewScheme()
	if err := scheme.AddToScheme(s); err != nil {
		return nil, err
	}

	return client.New(mgr.GetConfig(), client.Options{Scheme: s})
}
