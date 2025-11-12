// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package downloaders

import (
	"encoding/json"
	"fmt"

	"github.com/gardener/landscaper/legacy-component-cli/ociclient"
	"github.com/gardener/landscaper/legacy-component-cli/ociclient/cache"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process/extensions"
)

const (
	// LocalOCIBlobDownloaderType defines the type of a local oci blob downloader
	LocalOCIBlobDownloaderType = "LocalOciBlobDownloader"

	// OCIArtifactDownloaderType defines the type of an oci artifact downloader
	OCIArtifactDownloaderType = "OciArtifactDownloader"
)

// NewDownloaderFactory creates a new downloader factory
// How to add a new downloader (without using extension mechanism):
// - Add Go file to downloader package which contains the source code of the new downloader
// - Add string constant for new downloader type -> will be used in DownloaderFactory.Create()
// - Add source code for creating new downloader to DownloaderFactory.Create() method
func NewDownloaderFactory(client ociclient.Client, ocicache cache.Cache) *DownloaderFactory {
	return &DownloaderFactory{
		client: client,
		cache:  ocicache,
	}
}

// DownloaderFactory defines a helper struct for creating downloaders
type DownloaderFactory struct {
	client ociclient.Client
	cache  cache.Cache
}

// Create creates a new downloader defined by a type and a spec
func (f *DownloaderFactory) Create(downloaderType string, spec *json.RawMessage) (process.ResourceStreamProcessor, error) {
	switch downloaderType {
	case LocalOCIBlobDownloaderType:
		return NewLocalOCIBlobDownloader(f.client)
	case OCIArtifactDownloaderType:
		return NewOCIArtifactDownloader(f.client, f.cache)
	case extensions.ExecutableType:
		return extensions.CreateExecutable(spec)
	default:
		return nil, fmt.Errorf("unknown downloader type %s", downloaderType)
	}
}
