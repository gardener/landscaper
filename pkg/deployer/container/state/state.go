// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package state

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/container"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	containeractuator "github.com/gardener/landscaper/pkg/deployer/container"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/utils"
)

// State handles the backup and restore of state of container deploy item.
type State struct {
	log     logr.Logger
	backoff wait.Backoff

	deployItem lsv1alpha1.ObjectReference
	kubeClient client.Client
	path       string
}

// New creates a new state instance.
func New(log logr.Logger, kubeClient client.Client, deployItemKey lsv1alpha1.ObjectReference, statePath string) *State {
	return &State{
		log:        log,
		deployItem: deployItemKey,
		kubeClient: kubeClient,
		path:       statePath,
		backoff: wait.Backoff{
			Duration: 10 * time.Second,
			Factor:   1.2,
			Jitter:   0,
			Steps:    math.MaxInt32,
			Cap:      10,
		},
	}
}

// WithBackoff configures the state to use a spoecific backoff
func (s *State) WithBackoff(backoff wait.Backoff) *State {
	s.backoff = backoff
	return s
}

// Backup tars the content of the State directory and stores it in a secrets in the cluster.
func (s *State) Backup(ctx context.Context) error {
	// do nothing if there is no State to persist
	files, err := ioutil.ReadDir(s.path)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		s.log.Info("no State to persist")
		return nil
	}

	// tar and gzip the State content

	// todo: check if it would be possible to have the whole State in memory.
	tmpFile, err := ioutil.TempFile(os.TempDir(), "state-")
	if err != nil {
		return err
	}
	if err := utils.BuildTarGzip(osfs.New(), s.path, tmpFile); err != nil {
		tmpFile.Close()
		return errors.Wrap(err, "unable to tar and gzip State")
	}
	tmpFile.Close()
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			s.log.Error(err, "unable to remove tmp State file")
		}
	}()

	// split the file in chunks of 1MB (Secret size limit)
	secrets, err := s.splitFileAndUploadChunks(ctx, tmpFile.Name())
	if err != nil {
		return err
	}

	return wait.ExponentialBackoff(s.backoff, func() (bool, error) {
		deployItem := &lsv1alpha1.DeployItem{}
		if err := s.kubeClient.Get(ctx, s.deployItem.NamespacedName(), deployItem); err != nil {
			if apierrors.IsNotFound(err) {
				return false, err
			}
			s.log.Error(err, "unable to get deploy item")
			return false, nil
		}
		// update the provider status
		status, err := containeractuator.DecodeProviderStatus(deployItem.Status.ProviderStatus)
		if err != nil {
			return false, err
		}
		status.State = &containerv1alpha1.StateStatus{Data: secrets}
		encStatus, err := containeractuator.EncodeProviderStatus(status)
		if err != nil {
			return false, err
		}
		deployItem.Status.ProviderStatus = encStatus
		if err := s.kubeClient.Status().Update(ctx, deployItem); err != nil {
			if apierrors.IsNotFound(err) {
				return false, err
			}
			s.log.Error(err, "unable to update deploy item")
			return false, nil
		}
		return true, nil
	})
}

// Restore restores the latest state from the k8s cluster to the configured state path.
func (s *State) Restore(ctx context.Context, fs vfs.FileSystem) error {
	secretList := &corev1.SecretList{}
	labels := client.MatchingLabels{
		container.ContainerDeployerNameLabel: s.deployItem.Name,
		container.ContainerDeployerTypeLabel: "state",
	}
	if err := s.kubeClient.List(ctx, secretList, labels, client.InNamespace(s.deployItem.Namespace)); err != nil {
		return nil
	}

	// the secrets are grouped by uuid and sorted by their creation date
	secrets := map[string][]*corev1.Secret{}
	var newest *corev1.Secret
	for _, secret := range secretList.Items {
		if newest == nil || newest.CreationTimestamp.Before(&secret.CreationTimestamp) {
			newest = &secret
		}
		uuidStr := secret.Annotations[container.ContainerDeployerStateUUIDAnnotation]
		secrets[uuidStr] = append(secrets[uuidStr], &secret)
	}
	if newest == nil {
		return nil
	}
	newestUuid := newest.Annotations[container.ContainerDeployerStateUUIDAnnotation]
	if err := s.restoreFromSecrets(secrets[newestUuid], fs); err != nil {
		return err
	}

	// garbage collect the old states
	wg := sync.WaitGroup{}
	for uuidStr, sc := range secrets {
		if uuidStr == newestUuid {
			continue
		}
		wg.Add(1)
		go func(ctx context.Context, secrets []*corev1.Secret) {
			defer wg.Done()
			s.gcOldSecrets(ctx, secrets)
		}(ctx, sc)
	}
	wg.Wait()

	return nil
}

func (s *State) restoreFromSecrets(secrets []*corev1.Secret, fs vfs.FileSystem) error {
	sort.Sort(stateSecretsList(secrets))

	var data bytes.Buffer
	for _, secret := range secrets {
		chunk, ok := secret.Data[lsv1alpha1.DataObjectSecretDataKey]
		if !ok {
			return fmt.Errorf("expected chunk in secret %s", secret.Name)
		}
		data.Write(chunk)
	}

	return utils.ExtractTarGzip(&data, fs, s.path)
}

func (s *State) gcOldSecrets(ctx context.Context, secrets []*corev1.Secret) {
	for _, secret := range secrets {
		if err := s.kubeClient.Delete(ctx, secret); err != nil {
			s.log.Error(err, "unable to delete old state secret %s in namespace %s", secret.Name, secret.Namespace)
		}
	}
}

// splitFileAndUploadChunks splits the file given in the filepath into chunks of 1MB
// and uploads the chunks as secrets to the configured k8s cluster as secrets
func (s *State) splitFileAndUploadChunks(ctx context.Context, filePath string) ([]lsv1alpha1.ObjectReference, error) {
	const bufSize = corev1.MaxSecretSize // 1 MB
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	deployItem := &lsv1alpha1.DeployItem{}
	if err := s.kubeClient.Get(ctx, s.deployItem.NamespacedName(), deployItem); err != nil {
		return nil, err
	}

	secrets := make([]lsv1alpha1.ObjectReference, 0)
	uuidString := uuid.New().String()
	buffer := make([]byte, bufSize)
	count := 0
	for {
		if _, err := file.Read(buffer); err != nil {
			if err == io.EOF {
				return secrets, nil
			}
			return nil, err
		}

		secret := &corev1.Secret{}
		secret.GenerateName = s.deployItem.Name + "-"
		secret.Namespace = s.deployItem.Namespace
		secret.Labels = map[string]string{
			container.ContainerDeployerNameLabel: s.deployItem.Name,
			container.ContainerDeployerTypeLabel: "state", // todo: make const
		}
		secret.Annotations = map[string]string{
			container.ContainerDeployerStateUUIDAnnotation: uuidString,
			container.ContainerDeployerStateNumAnnotation:  strconv.Itoa(count),
		}
		secret.Data = map[string][]byte{
			lsv1alpha1.DataObjectSecretDataKey: buffer,
		}
		if err := controllerutil.SetControllerReference(deployItem, secret, kubernetes.LandscaperScheme); err != nil {
			return secrets, err
		}

		if err := s.kubeClient.Create(ctx, secret); err != nil {
			return secrets, err
		}
		secrets = append(secrets, lsv1alpha1.ObjectReference{
			Name:      secret.Name,
			Namespace: secret.Namespace,
		})
		count++
	}
}

type stateSecretsList []*corev1.Secret

func (s stateSecretsList) Len() int { return len(s) }

func (s stateSecretsList) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s stateSecretsList) Less(i, j int) bool {
	numI, _ := strconv.Atoi(s[i].Annotations[container.ContainerDeployerStateNumAnnotation])
	numJ, _ := strconv.Atoi(s[j].Annotations[container.ContainerDeployerStateNumAnnotation])
	return numI < numJ
}
