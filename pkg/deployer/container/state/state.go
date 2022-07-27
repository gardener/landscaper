// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package state

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/utils/tar"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/container"
)

// State handles the backup and restore of state of container deploy item.
type State struct {
	log logging.Logger

	deployItem lsv1alpha1.ObjectReference
	// namespace is the namespace where the state secrets should be created.
	namespace  string
	kubeClient client.Client
	fs         vfs.FileSystem
	path       string
}

// New creates a new state instance.
func New(log logging.Logger, kubeClient client.Client, namespace string, deployItemKey lsv1alpha1.ObjectReference, statePath string) *State {
	return &State{
		log:        log,
		deployItem: deployItemKey,
		namespace:  namespace,
		kubeClient: kubeClient,
		fs:         osfs.New(),
		path:       statePath,
	}
}

// WithFs sets the fs for the state
func (s *State) WithFs(fs vfs.FileSystem) *State {
	s.fs = fs
	return s
}

// Backup tars the content of the State directory and stores it in a secrets in the cluster.
func (s *State) Backup(ctx context.Context) error {
	// do nothing if there is no State to persist
	files, err := vfs.ReadDir(s.fs, s.path)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		s.log.Info("no State to persist")
		return nil
	}

	// tar and gzip the State content

	// todo: check if it would be possible to have the whole State in memory.
	tmpFile, err := vfs.TempFile(s.fs, s.fs.FSTempDir(), "state-")
	if err != nil {
		return err
	}
	if err := tar.BuildTarGzip(s.fs, s.path, tmpFile); err != nil {
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
	_, err = s.splitFileAndUploadChunks(ctx, tmpFile.Name())
	if err != nil {
		return err
	}
	return nil
}

// StateSecretListOptions returns the list options for all state secrets of a deploy item
func StateSecretListOptions(namespace string, deployItem lsv1alpha1.ObjectReference) []client.ListOption {
	labelSelector := client.MatchingLabels{
		container.ContainerDeployerDeployItemNameLabel:      deployItem.Name,
		container.ContainerDeployerDeployItemNamespaceLabel: deployItem.Namespace,
		container.ContainerDeployerTypeLabel:                "state",
	}
	return []client.ListOption{labelSelector, client.InNamespace(namespace)}
}

// Restore restores the latest state from the k8s cluster to the configured state path.
func (s *State) Restore(ctx context.Context) error {
	if len(s.deployItem.Name) == 0 || len(s.deployItem.Namespace) == 0 {
		return fmt.Errorf("a deployitem has to be defined")
	}
	if len(s.namespace) == 0 {
		return fmt.Errorf("a target namespace has to be defined")
	}

	if _, err := s.fs.Stat(s.path); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("unable to read from filesystem: %w", err)
		}
		if err := s.fs.MkdirAll(s.path, os.ModePerm); err != nil {
			return fmt.Errorf("unable to create directory %q: %w", s.path, err)
		}
	}

	secretList := &corev1.SecretList{}
	if err := s.kubeClient.List(ctx, secretList, StateSecretListOptions(s.namespace, s.deployItem)...); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	// the secrets are grouped by uuid and sorted by their creation date
	s.log.Info(fmt.Sprintf("Restore state from %d secrets", len(secretList.Items)))
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
	if err := s.restoreFromSecrets(secrets[newestUuid]); err != nil {
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

func (s *State) restoreFromSecrets(secrets []*corev1.Secret) error {
	sort.Sort(stateSecretsList(secrets))

	// todo: need to write to filesystem
	var data bytes.Buffer
	for _, secret := range secrets {
		chunk, ok := secret.Data[lsv1alpha1.DataObjectSecretDataKey]
		if !ok {
			return fmt.Errorf("expected chunk in secret %s", secret.Name)
		}
		data.Write(chunk)
	}

	return tar.ExtractTarGzip(context.TODO(), &data, s.fs, tar.ToPath(s.path))
}

func (s *State) gcOldSecrets(ctx context.Context, secrets []*corev1.Secret) {
	for _, secret := range secrets {
		if err := s.kubeClient.Delete(ctx, secret); err != nil {
			s.log.Error(err, "unable to delete old state secret %s in namespace %s", secret.Name, secret.Namespace)
		}
		s.log.Info(fmt.Sprintf("Successfully garbage collected %q", secret.Name))
	}
}

// splitFileAndUploadChunks splits the file given in the filepath into chunks of 1MB
// and uploads the chunks as secrets to the configured k8s cluster as secrets
func (s *State) splitFileAndUploadChunks(ctx context.Context, filePath string) ([]lsv1alpha1.ObjectReference, error) {
	const bufSize = corev1.MaxSecretSize // 1 MB
	file, err := s.fs.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

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
		// remove NULL characters
		buffer = bytes.Trim(buffer, "\x00")

		secret := &corev1.Secret{}
		secret.GenerateName = fmt.Sprintf("state-%s-%s-", s.deployItem.Namespace, s.deployItem.Name)
		secret.Namespace = s.namespace
		secret.Labels = map[string]string{
			container.ContainerDeployerDeployItemNameLabel:      s.deployItem.Name,
			container.ContainerDeployerDeployItemNamespaceLabel: s.deployItem.Namespace,
			container.ContainerDeployerTypeLabel:                "state", // todo: make const
		}
		secret.Annotations = map[string]string{
			container.ContainerDeployerStateUUIDAnnotation: uuidString,
			container.ContainerDeployerStateNumAnnotation:  strconv.Itoa(count),
		}
		secret.Data = map[string][]byte{
			lsv1alpha1.DataObjectSecretDataKey: buffer,
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

// CleanupState deletes all state secrets for a deployitem
func CleanupState(ctx context.Context, log logging.Logger, kubeClient client.Client, namespace string, deployItem lsv1alpha1.ObjectReference) error {
	secretList := &corev1.SecretList{}
	if err := kubeClient.List(ctx, secretList, StateSecretListOptions(namespace, deployItem)...); err != nil {
		return nil
	}

	bo := wait.Backoff{
		Duration: 10 * time.Second,
		Factor:   1.2,
		Jitter:   0,
		Steps:    math.MaxInt32,
		Cap:      10 * time.Minute,
	}
	return wait.ExponentialBackoff(bo, func() (done bool, err error) {
		completed := true
		for _, secret := range secretList.Items {
			if err := kubeClient.Delete(ctx, &secret); err != nil {
				if apierrors.IsNotFound(err) {
					continue
				}
				log.Error(err, "unable to delete state secret")
			}
			completed = false
		}
		return completed, nil
	})
}
