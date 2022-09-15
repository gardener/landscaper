// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package pkg

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

	"github.com/gardener/landscaper/hack/testcluster/pkg/utils"

	"github.com/docker/cli/cli/config/configfile"
	dockerconfigtypes "github.com/docker/cli/cli/config/types"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/webhook/certificates"
)

const (
	AddressFormatHostname = "hostname"
	AddressFormatIP       = "ip"
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

func CreateRegistry(ctx context.Context,
	logger utils.Logger,
	kubeClient client.Client,
	restConfig *rest.Config,
	namespace string,
	id string,
	stateFile string,
	password string,
	outputAddressFormat string,
	exportRegistryCreds string,
	timeout time.Duration,
	runOnShoot bool) (err error) {

	logger.Logln("Creating registry service")
	svc := &corev1.Service{}
	svc.Name = "registry-" + id
	svc.Namespace = namespace
	kutil.SetMetaDataLabel(svc, ClusterIdLabelName, id)
	kutil.SetMetaDataLabel(svc, ComponentLabelName, ComponentRegistry)
	svc.Spec.Ports = []corev1.ServicePort{
		{
			Name:       "registry",
			Port:       5000,
			TargetPort: intstr.FromInt(5000),
		},
	}
	svc.Spec.Selector = svc.Labels

	if runOnShoot {
		svc.Spec.Type = corev1.ServiceTypeLoadBalancer
	}
	if err := kubeClient.Create(ctx, svc); err != nil {
		return err
	}
	logger.Logln("Successfully created registry service")

	// register cleanup to delete the cluster if something fails
	defer func() {
		if err == nil {
			return
		}
		if err := cleanupRegistry(ctx,
			logger,
			kubeClient,
			namespace,
			id,
			timeout); err != nil {
			logger.Logfln("Error while cleanup of the registry: %s", err.Error())
		}
	}()

	logger.Logln("Create registry certificates")
	certSecret, err := generateCertificate(svc)
	if err != nil {
		return fmt.Errorf("unable to create certificates for registry: %w", err)
	}
	if err := kubeClient.Create(ctx, certSecret); err != nil {
		return err
	}
	logger.Logln("Successfully created registry certificates")

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
		logger.Log(podBytes.String())
		return fmt.Errorf("unable to decode pod: %w", err)
	}
	pod.Name = svc.Name
	pod.Namespace = namespace
	pod.Labels = svc.Labels

	if err := kubeClient.Create(ctx, pod); err != nil {
		return fmt.Errorf("unable to create registry pod: %w", err)
	}
	logger.Logfln("Created registry %q", pod.Name)

	err = wait.PollImmediate(10*time.Second, timeout, func() (done bool, err error) {
		updatedPod := &corev1.Pod{}
		if err := kubeClient.Get(ctx, client.ObjectKey{Name: pod.Name, Namespace: pod.Namespace}, updatedPod); err != nil {
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
	if len(stateFile) != 0 {
		if err := os.MkdirAll(filepath.Dir(stateFile), os.ModePerm); err != nil {
			return fmt.Errorf("unable to create state directory %q: %w", filepath.Dir(stateFile), err)
		}

		data, err := json.Marshal(State{ID: id})
		if err != nil {
			return fmt.Errorf("unable to marshal state: %w", err)
		}
		if err := ioutil.WriteFile(stateFile, data, os.ModePerm); err != nil {
			return fmt.Errorf("unable to write statefile to %q: %w", stateFile, err)
		}
		logger.Logfln("Successfully written state to %q", stateFile)
	}

	auth := dockerconfigtypes.AuthConfig{
		ServerAddress: fmt.Sprintf("%s.%s:5000", svc.Name, svc.Namespace),
		Username:      username,
		Password:      password,
		Auth:          base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password))),
	}
	switch outputAddressFormat {
	case AddressFormatHostname:
		auth.ServerAddress = fmt.Sprintf("%s.%s:5000", svc.Name, svc.Namespace)
	case AddressFormatIP:
		if runOnShoot {
			err = wait.PollImmediate(10*time.Second, 2*time.Minute, func() (done bool, err error) {
				tmpSvc := &corev1.Service{}
				if tmpErr := kubeClient.Get(ctx, client.ObjectKeyFromObject(svc), tmpSvc); tmpErr != nil {
					return false, tmpErr
				}
				if len(tmpSvc.Status.LoadBalancer.Ingress) > 0 && len(tmpSvc.Status.LoadBalancer.Ingress[0].IP) > 0 {
					auth.ServerAddress = fmt.Sprintf("%s:5000", tmpSvc.Status.LoadBalancer.Ingress[0].IP)
					logger.Logln(fmt.Sprintf("External IP detected: %s", auth.ServerAddress))
					return true, nil
				}
				return false, nil
			})
		} else {
			auth.ServerAddress = fmt.Sprintf("%s:5000", svc.Spec.ClusterIP)
		}

		logger.Logln(fmt.Sprintf("IP detected: %s", auth.ServerAddress))

	default:
		return fmt.Errorf("unknown address format %q", outputAddressFormat)
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

	if len(exportRegistryCreds) == 0 {
		logger.Logfln("password: \n%q", string(dockerconfigBytes))
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(exportRegistryCreds), os.ModePerm); err != nil {
		return fmt.Errorf("unable to create export directory %q: %w", filepath.Dir(exportRegistryCreds), err)
	}
	if err := ioutil.WriteFile(exportRegistryCreds, dockerconfigBytes, os.ModePerm); err != nil {
		return fmt.Errorf("unable to write docker auth config to %q: %w", exportRegistryCreds, err)
	}
	logger.Logfln("Successfully written docker auth config to %q", exportRegistryCreds)

	return nil
}

// DeleteRegistry deletes a previously created registry.
func DeleteRegistry(ctx context.Context,
	logger utils.Logger,
	kubeClient client.Client,
	namespace string,
	id string,
	timeout time.Duration) error {
	if len(id) == 0 {
		return errors.New("no id defined by flag or statefile")
	}

	if err := cleanupRegistry(ctx,
		logger,
		kubeClient,
		namespace,
		id,
		timeout); err != nil {
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
func cleanupRegistry(ctx context.Context,
	logger utils.Logger,
	kubeClient client.Client,
	namespace string,
	id string,
	timeout time.Duration) error {

	podList := &corev1.PodList{}
	if err := kubeClient.List(ctx, podList, client.InNamespace(namespace), client.MatchingLabels{
		ClusterIdLabelName: id,
		ComponentLabelName: ComponentRegistry,
	}); err != nil {
		return fmt.Errorf("unable to get pods for id %q in namespace %q: %w", id, namespace, err)
	}

	if len(podList.Items) == 0 {
		logger.Logln("No pods found")
	}
	for _, pod := range podList.Items {
		if err := cleanupObject(ctx, logger, ComponentRegistry, kubeClient, &pod, timeout); err != nil {
			return err
		}
	}

	logger.Logfln("Cleanup registry secrets")
	secretList := &corev1.SecretList{}
	if err := kubeClient.List(ctx, secretList, client.InNamespace(namespace), client.MatchingLabels{
		ClusterIdLabelName: id,
		ComponentLabelName: ComponentRegistry,
	}); err != nil {
		return fmt.Errorf("unable to get secrets for id %q in namespace %q: %w", id, namespace, err)
	}
	if len(secretList.Items) == 0 {
		logger.Logln("No secrets found")
	}
	for _, secret := range secretList.Items {
		if err := cleanupObject(ctx, logger, ComponentRegistry, kubeClient, &secret, timeout); err != nil {
			return err
		}
	}

	logger.Logfln("Cleanup registry services")
	svcList := &corev1.ServiceList{}
	if err := kubeClient.List(ctx, svcList, client.InNamespace(namespace), client.MatchingLabels{
		ClusterIdLabelName: id,
		ComponentLabelName: ComponentRegistry,
	}); err != nil {
		return fmt.Errorf("unable to get services for id %q in namespace %q: %w", id, namespace, err)
	}
	if len(svcList.Items) == 0 {
		logger.Logln("No services found")
	}
	for _, svc := range svcList.Items {
		if err := cleanupObject(ctx, logger, ComponentRegistry, kubeClient, &svc, timeout); err != nil {
			return err
		}
	}
	return nil
}

// cleanupObject deletes the objects.
func cleanupObject(ctx context.Context, logger utils.Logger, componentName string, kubeClient client.Client, obj client.Object, timeout time.Duration) error {
	gvk, err := apiutil.GVKForObject(obj, scheme.Scheme)
	if err != nil {
		return fmt.Errorf("unable to get group version kind for %s/%s: %w", obj.GetNamespace(), obj.GetName(), err)
	}
	objDesc := fmt.Sprintf("%q %q of %s", gvk.String(), obj.GetName(), componentName)
	logger.Logfln("Cleanup %s", objDesc)
	err = wait.PollImmediate(10*time.Second, 1*time.Minute, func() (done bool, err error) {
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
