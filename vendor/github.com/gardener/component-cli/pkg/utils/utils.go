// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/ociclient/oci"
	"github.com/gardener/component-cli/pkg/commands/constants"
)

// PrintPrettyYaml prints the given objects as yaml if enabled.
func PrintPrettyYaml(obj interface{}, enabled bool) {
	if !enabled {
		return
	}
	data, err := yaml.Marshal(obj)
	if err != nil {
		fmt.Printf("unable to serialize object as yaml: %s", err.Error())
		return
	}
	fmt.Println(string(data))
}

// GetFileType returns the mimetype of a file.
func GetFileType(fs vfs.FileSystem, path string) (string, error) {
	file, err := fs.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	// see http://golang.org/pkg/net/http/#DetectContentType for the 512 bytes
	buf := make([]byte, 512)
	_, err = file.Read(buf)
	if err != nil {
		return "", err
	}
	return http.DetectContentType(buf), nil
}

// CacheDir returns the cache dir for the current cli command
func CacheDir() (string, error) {
	defaultCacheDir := os.Getenv(cache.CacheDirEnvName)
	if len(defaultCacheDir) != 0 {
		return defaultCacheDir, nil
	}

	cliHomeDir, err := constants.CliHomeDir()
	if err != nil {
		return "", err
	}
	cacheDir := filepath.Join(cliHomeDir, "components")
	if err := os.MkdirAll(cacheDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("unable to create cache directory %s: %w", cacheDir, err)
	}

	return cacheDir, nil
}

// CleanMarkdownUsageFunc removes markdown tags from the long usage of the command.
// With this func it is possible to generate the markdown docs but still have readable commandline help func.
// Note: currently only "<pre>" tags are removed
func CleanMarkdownUsageFunc(cmd *cobra.Command) {
	defaultHelpFunc := cmd.HelpFunc()
	cmd.SetHelpFunc(func(cmd *cobra.Command, s []string) {
		cmd.Long = strings.ReplaceAll(cmd.Long, "<pre>", "")
		cmd.Long = strings.ReplaceAll(cmd.Long, "</pre>", "")
		defaultHelpFunc(cmd, s)
	})
}

// RawJSON converts an arbitrary value to json.RawMessage
func RawJSON(value interface{}) (*json.RawMessage, error) {
	jsonval, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return (*json.RawMessage)(&jsonval), nil
}

// Gzip applies gzip compression to an arbitrary byte slice
func Gzip(data []byte, compressionLevel int) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	gzipWriter, err := gzip.NewWriterLevel(buf, compressionLevel)
	if err != nil {
		return nil, fmt.Errorf("unable to create gzip writer: %w", err)
	}
	defer gzipWriter.Close()

	if _, err = gzipWriter.Write(data); err != nil {
		return nil, fmt.Errorf("unable to write to stream: %w", err)
	}

	if err = gzipWriter.Close(); err != nil {
		return nil, fmt.Errorf("unable to close writer: %w", err)
	}

	return buf.Bytes(), nil
}

var chars = []rune("abcdefghijklmnopqrstuvwxyz1234567890")

// RandomString creates a new random string with the given length.
func RandomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

// SafeConvert converts a byte slice to string.
// If the byte slice is nil, an empty string is returned.
func SafeConvert(bytes []byte) string {
	if bytes == nil {
		return ""
	}

	return string(bytes)
}

const (
	BYTE = 1.0 << (10 * iota)
	KIBIBYTE
	MEBIBYTE
	GIBIBYTE
)

// BytesString converts bytes into a human readable string.
// This function is inspired by https://www.reddit.com/r/golang/comments/8micn7/review_bytes_to_human_readable_format/
func BytesString(bytes uint64, accuracy int) string {
	unit := ""
	value := float32(bytes)

	switch {
	case bytes >= GIBIBYTE:
		unit = "GiB"
		value = value / GIBIBYTE
	case bytes >= MEBIBYTE:
		unit = "MiB"
		value = value / MEBIBYTE
	case bytes >= KIBIBYTE:
		unit = "KiB"
		value = value / KIBIBYTE
	case bytes >= BYTE:
		unit = "bytes"
	case bytes == 0:
		return "0"
	}

	stringValue := strings.TrimSuffix(
		fmt.Sprintf("%.2f", value), "."+strings.Repeat("0", accuracy),
	)

	return fmt.Sprintf("%s %s", stringValue, unit)
}

// WriteFileToTARArchive writes a new file with name=filename and content=inputReader to outputWriter
func WriteFileToTARArchive(filename string, inputReader io.Reader, outputWriter *tar.Writer) error {
	if filename == "" {
		return errors.New("filename must not be empty")
	}

	if inputReader == nil {
		return errors.New("inputReader must not be nil")
	}

	if outputWriter == nil {
		return errors.New("outputWriter must not be nil")
	}

	tempfile, err := ioutil.TempFile("", "")
	if err != nil {
		return fmt.Errorf("unable to create tempfile: %w", err)
	}
	defer tempfile.Close()

	fsize, err := io.Copy(tempfile, inputReader)
	if err != nil {
		return fmt.Errorf("unable to copy content to tempfile: %w", err)
	}

	if _, err := tempfile.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("unable to seek to beginning of tempfile: %w", err)
	}

	header := tar.Header{
		Name:    filename,
		Size:    int64(fsize),
		Mode:    0600,
		ModTime: time.Now(),
	}

	if err := outputWriter.WriteHeader(&header); err != nil {
		return fmt.Errorf("unable to write tar header: %w", err)
	}

	if _, err := io.Copy(outputWriter, tempfile); err != nil {
		return fmt.Errorf("unable to write file to tar archive: %w", err)
	}

	return nil
}

// TargetOCIArtifactRef calculates the target reference for
func TargetOCIArtifactRef(targetRepo, ref string, keepOrigHost bool) (string, error) {
	if !strings.Contains(targetRepo, "://") {
		// add dummy protocol to correctly parse the url
		targetRepo = "http://" + targetRepo
	}
	t, err := url.Parse(targetRepo)
	if err != nil {
		return "", err
	}
	parsedRef, err := oci.ParseRef(ref)
	if err != nil {
		return "", err
	}

	if !keepOrigHost {
		parsedRef.Host = t.Host
		parsedRef.Repository = path.Join(t.Path, parsedRef.Repository)
		return parsedRef.String(), nil
	}
	replacedRef := strings.NewReplacer(".", "_", ":", "_").Replace(parsedRef.Name())
	parsedRef.Repository = path.Join(t.Path, replacedRef)
	parsedRef.Host = t.Host
	return parsedRef.String(), nil
}

// CalculateBlobUploadRef calculates the OCI reference where blobs for a component should be uploaded
func CalculateBlobUploadRef(repoCtx cdv2.OCIRegistryRepository, componentName string, componentVersion string) string {
	uploadTag := componentVersion
	if strings.Contains(componentVersion, ":") {
		// if componentVersion is a digest, use "latest" tag for upload ref
		uploadTag = "latest"
	}

	return fmt.Sprintf("%s/component-descriptors/%s:%s", repoCtx.BaseURL, componentName, uploadTag)
}

// ExponentialBackoff calculates an exponential backoff timeout
func ExponentialBackoff(backoffFactor time.Duration, retries uint64) time.Duration {
	return time.Duration(float64(backoffFactor.Nanoseconds()) * math.Pow(2, float64(retries)))
}
