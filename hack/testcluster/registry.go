// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/docker/cli/cli/config/configfile"
	dockerconfigtypes "github.com/docker/cli/cli/config/types"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
	"github.com/gardener/landscaper/pkg/utils/simplelogger"
	"github.com/gardener/landscaper/pkg/utils/webhook/certificates"
)

// see this issue for more discussions around min resources for a k8s cluster running in a pod.
// we will stick for now with req of 500M RAM/1 CPU
const registryPodTmpl = `
apiVersion: v1
kind: Pod
metadata:
  name: registry
spec:
  containers:
  - image: registry:2
    imagePullPolicy: IfNotPresent
    name: registry
    env:
    - name: REGISTRY_AUTH
      value: "htpasswd"
    - name: REGISTRY_AUTH_HTPASSWD_REALM
      value: "Registry Realm"
    - name: REGISTRY_AUTH_HTPASSWD_PATH
      value: "/shared/htpasswd"
    - name: REGISTRY_HTTP_TLS_CERTIFICATE
      value: "/certs/tls.crt"
    - name: REGISTRY_HTTP_TLS_KEY
      value: "/certs/tls.key"
    - name: POD_IP
      valueFrom:
        fieldRef:
          fieldPath: status.podIP
    securityContext:
      privileged: true
    ports:
    - containerPort: 5000
      name: registry
      protocol: TCP
    resources:
      requests:
        memory: "256M"
        cpu: "100m"
      limits:
        memory: "1G"
        cpu: "1"
    volumeMounts:
    - name: shared-data
      mountPath: /shared
    - name: certs-vol
      mountPath: "/certs"
      readOnly: true
    readinessProbe:
      failureThreshold: 15
      httpGet:
        path: /
        port: registry
        scheme: HTTPS
      initialDelaySeconds: 5
      periodSeconds: 20
      successThreshold: 1
      timeoutSeconds: 1
  initContainers:
  - name: set-password
    image: busybox:1.28
    command: ['sh', '-c', "echo '{{ .htpasswd }}' > /shared/htpasswd"]
    volumeMounts:
    - name: shared-data
      mountPath: /shared
  volumes:
  - name: shared-data
    emptyDir: {}
  - name: certs-vol
    secret:
      secretName: {{ .secretName }}
`

func createRegistry(ctx context.Context, logger simplelogger.Logger, opts *Options) (err error) {
	if !opts.EnableRegistry {
		return nil
	}

	logger.Logln("Creating registry service")
	svc := &corev1.Service{}
	svc.Name = "registry-" + opts.ID
	svc.Namespace = opts.Namespace
	kutil.SetMetaDataLabel(svc, ClusterIdLabelName, opts.ID)
	kutil.SetMetaDataLabel(svc, ComponentLabelName, ComponentRegistry)
	svc.Spec.Ports = []corev1.ServicePort{
		{
			Name:       "registry",
			Port:       5000,
			TargetPort: intstr.FromInt(5000),
		},
	}
	svc.Spec.Selector = svc.Labels
	if err := opts.kubeClient.Create(ctx, svc); err != nil {
		return err
	}
	logger.Logln("Successfully created registry service")

	// register cleanup to delete the cluster if something fails
	defer func() {
		if err == nil {
			return
		}
		if err := cleanupRegistry(ctx, logger, opts); err != nil {
			logger.Logfln("Error while cleanup of the registry: %s", err.Error())
		}
	}()

	logger.Logln("Create registry certificates")
	certSecret, err := generateCertificate(svc)
	if err != nil {
		return fmt.Errorf("unable to create certificates for registry: %w", err)
	}
	if err := opts.kubeClient.Create(ctx, certSecret); err != nil {
		return err
	}
	logger.Logln("Successfully created registry certificates")

	password := opts.Password
	username := "test"
	htpasswd := CreateHtpasswd(username, password)

	// parse and template pod
	tmpl, err := template.New("pod").Parse(registryPodTmpl)
	if err != nil {
		return fmt.Errorf("unable to registry pod template: %w", err)
	}
	var podBytes bytes.Buffer
	if err := tmpl.Execute(&podBytes, map[string]interface{}{
		"htpasswd":   htpasswd,
		"secretName": svc.Name,
	}); err != nil {
		return err
	}

	pod := &corev1.Pod{}
	if _, _, err := serializer.NewCodecFactory(scheme.Scheme).UniversalDecoder().Decode(podBytes.Bytes(), nil, pod); err != nil {
		logger.Log(string(podBytes.Bytes()))
		return fmt.Errorf("unable to decode pod: %w", err)
	}
	pod.Name = svc.Name
	pod.Namespace = opts.Namespace
	pod.Labels = svc.Labels

	if err := opts.kubeClient.Create(ctx, pod); err != nil {
		return fmt.Errorf("unable to create registry pod: %w", err)
	}
	logger.Logfln("Created registry %q", pod.Name)

	err = wait.PollImmediate(10*time.Second, opts.Timeout, func() (done bool, err error) {
		updatedPod := &corev1.Pod{}
		if err := opts.kubeClient.Get(ctx, client.ObjectKey{Name: pod.Name, Namespace: pod.Namespace}, updatedPod); err != nil {
			return false, err
		}
		*pod = *updatedPod

		if updatedPod.Status.Phase != corev1.PodRunning {
			logger.Logln("Waiting for registry pod to be up and running...")
			return false, nil
		}

		// check pod status
		if len(updatedPod.Status.ContainerStatuses) == 0 {
			logger.Logln("Waiting for registry pod to be up and running...")
			return false, nil
		}
		if !updatedPod.Status.ContainerStatuses[0].Ready {
			logger.Logln("Waiting for registry to be ready...")
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return err
	}
	logger.Logln("Successfully created registry pod")

	logger.Logln("Successfully created registry")

	// write state if path is provided
	if len(opts.StateFile) != 0 {
		if err := os.MkdirAll(filepath.Dir(opts.StateFile), os.ModePerm); err != nil {
			return fmt.Errorf("unable to create state directory %q: %w", filepath.Dir(opts.StateFile), err)
		}

		data, err := json.Marshal(State{ID: opts.ID})
		if err != nil {
			return fmt.Errorf("unable to marshal state: %w", err)
		}
		if err := ioutil.WriteFile(opts.StateFile, data, os.ModePerm); err != nil {
			return fmt.Errorf("unable to write statefile to %q: %w", opts.StateFile, err)
		}
		logger.Logfln("Successfully written state to %q", opts.StateFile)
	}

	auth := dockerconfigtypes.AuthConfig{
		ServerAddress: fmt.Sprintf("%s.%s:5000", svc.Name, svc.Namespace),
		Username:      username,
		Password:      password,
		Auth:          base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password))),
	}
	switch opts.OutputAddressFormat {
	case AddressFormatHostname:
		auth.ServerAddress = fmt.Sprintf("%s.%s:5000", svc.Name, svc.Namespace)
	case AddressFormatIP:
		auth.ServerAddress = fmt.Sprintf("%s:5000", svc.Spec.ClusterIP)
	default:
		return fmt.Errorf("unknown address format %q", opts.OutputAddressFormat)
	}
	dockerconfig := configfile.ConfigFile{
		AuthConfigs: map[string]dockerconfigtypes.AuthConfig{
			auth.ServerAddress: auth,
		},
	}

	dockerconfigBytes, err := json.Marshal(dockerconfig)
	if err != nil {
		return fmt.Errorf("unable to marshal docker config: %w", err)
	}

	if len(opts.ExportRegistryCreds) == 0 {
		logger.Logfln("password: \n%q", string(dockerconfigBytes))
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(opts.ExportRegistryCreds), os.ModePerm); err != nil {
		return fmt.Errorf("unable to create export directory %q: %w", filepath.Dir(opts.ExportRegistryCreds), err)
	}
	if err := ioutil.WriteFile(opts.ExportRegistryCreds, dockerconfigBytes, os.ModePerm); err != nil {
		return fmt.Errorf("unable to write docker auth config to %q: %w", opts.ExportRegistryCreds, err)
	}
	logger.Logfln("Successfully written docker auth config to %q", opts.ExportRegistryCreds)

	return nil
}

func deleteRegistry(ctx context.Context, logger simplelogger.Logger, opts *Options) error {
	if !opts.EnableRegistry {
		return nil
	}
	if len(opts.ID) == 0 {
		return errors.New("no id defined by flag or statefile")
	}

	if err := cleanupRegistry(ctx, logger, opts); err != nil {
		logger.Logfln("Error while cleanup of the registry: %s", err.Error())
	}

	return nil
}

func generateCertificate(svc *corev1.Service) (*corev1.Secret, error) {
	caConfig := certificates.CertificateSecretConfig{
		CertType: certificates.CACert,
		PKCS:     certificates.PKCS8,
	}
	caCert, err := caConfig.GenerateCertificate()
	if err != nil {
		return nil, err
	}

	dnsNames := []string{
		svc.Name,
		fmt.Sprintf("%s.%s", svc.Name, svc.Namespace),
		fmt.Sprintf("%s.%s.svc", svc.Name, svc.Namespace),
	}
	ipAddresses := []net.IP{
		net.ParseIP(svc.Spec.ClusterIP),
	}

	serverConfig := &certificates.CertificateSecretConfig{
		CommonName:  svc.Name,
		DNSNames:    dnsNames,
		IPAddresses: ipAddresses,
		CertType:    certificates.ServerCert,
		SigningCA:   caCert,
		PKCS:        caConfig.PKCS,
	}

	serverCert, err := serverConfig.GenerateCertificate()
	if err != nil {
		return nil, err
	}

	secret := &corev1.Secret{}
	secret.Name = svc.Name
	secret.Namespace = svc.Namespace
	secret.Labels = svc.Labels
	secret.Type = corev1.SecretTypeOpaque
	secret.Data = map[string][]byte{
		certificates.DataKeyCertificateCA: caCert.CertificatePEM,
		certificates.DataKeyPrivateKeyCA:  caCert.PrivateKeyPEM,
		certificates.DataKeyCertificate:   serverCert.CertificatePEM,
		certificates.DataKeyPrivateKey:    serverCert.PrivateKeyPEM,
	}
	return secret, nil
}

// cleanupRegistry deletes the registry and additional resources.
func cleanupRegistry(ctx context.Context, logger simplelogger.Logger, opts *Options) error {

	podList := &corev1.PodList{}
	if err := opts.kubeClient.List(ctx, podList, client.InNamespace(opts.Namespace), client.MatchingLabels{
		ClusterIdLabelName: opts.ID,
		ComponentLabelName: ComponentRegistry,
	}); err != nil {
		return fmt.Errorf("unable to get pods for id %q in namespace %q: %w", opts.ID, opts.Namespace, err)
	}

	for _, pod := range podList.Items {
		if err := cleanupObject(ctx, logger, ComponentRegistry, opts.kubeClient, &pod, opts.Timeout); err != nil {
			return err
		}
	}

	logger.Logfln("Cleanup registry secrets")
	secretList := &corev1.SecretList{}
	if err := opts.kubeClient.List(ctx, secretList, client.InNamespace(opts.Namespace), client.MatchingLabels{
		ClusterIdLabelName: opts.ID,
		ComponentLabelName: ComponentRegistry,
	}); err != nil {
		return fmt.Errorf("unable to get secrets for id %q in namespace %q: %w", opts.ID, opts.Namespace, err)
	}
	for _, secret := range secretList.Items {
		if err := cleanupObject(ctx, logger, ComponentRegistry, opts.kubeClient, &secret, opts.Timeout); err != nil {
			return err
		}
	}

	logger.Logfln("Cleanup registry services")
	svcList := &corev1.ServiceList{}
	if err := opts.kubeClient.List(ctx, svcList, client.InNamespace(opts.Namespace), client.MatchingLabels{
		ClusterIdLabelName: opts.ID,
		ComponentLabelName: ComponentRegistry,
	}); err != nil {
		return fmt.Errorf("unable to get services for id %q in namespace %q: %w", opts.ID, opts.Namespace, err)
	}
	for _, svc := range svcList.Items {
		if err := cleanupObject(ctx, logger, ComponentRegistry, opts.kubeClient, &svc, opts.Timeout); err != nil {
			return err
		}
	}
	return nil
}

// cleanupObject deletes the objects.
func cleanupObject(ctx context.Context, logger simplelogger.Logger, componentName string, kubeClient client.Client, obj client.Object, timeout time.Duration) error {
	objDesc := fmt.Sprintf("%q %q of %s", obj.GetObjectKind().GroupVersionKind().String(), obj.GetName(), componentName)
	logger.Logfln("Cleanup %s", objDesc)
	err := wait.PollImmediate(10*time.Second, 1*time.Minute, func() (done bool, err error) {
		if err := kubeClient.Delete(ctx, obj); err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			logger.Logfln("Error while trying to delete %s (%s)...", objDesc, err.Error())
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("unable to delete %s", objDesc)
	}
	err = wait.PollImmediate(10*time.Second, timeout, func() (done bool, err error) {
		updated := obj.DeepCopyObject().(client.Object)
		if err := kubeClient.Get(ctx, kutil.ObjectKey(obj.GetName(), obj.GetNamespace()), updated); err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			logger.Logfln("Error while trying to %s cluster (%s)...", objDesc, err.Error())
			return false, nil
		}
		logger.Logfln("Waiting for the %s to be deleted ...", objDesc)
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("unable to delete %s", objDesc)
	}
	logger.Logfln("Successfully deleted %s", objDesc)
	return nil
}
