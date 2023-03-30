// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/open-component-model/ocm/pkg/common/accessio/downloader"
)

// Downloader simply uses the default HTTP client to download the contents of a URL.
type Downloader struct {
	link string
}

func NewDownloader(link string) downloader.Downloader {
	return &Downloader{
		link: link,
	}
}

func (h *Downloader) Download(w io.WriterAt) error {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, h.link, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get link: %w", err)
	}
	defer resp.Body.Close()

	var blob []byte
	buf := bytes.NewBuffer(blob)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return fmt.Errorf("failed to copy response body: %w", err)
	}
	if _, err := w.WriteAt(buf.Bytes(), 0); err != nil {
		return fmt.Errorf("failed to WriteAt to the writer: %w", err)
	}
	return nil
}
