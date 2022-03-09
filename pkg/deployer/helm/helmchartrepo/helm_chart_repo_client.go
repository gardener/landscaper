package helmchartrepo

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/repo"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

const (
	defaultTimeoutSeconds = 180
	logLevelDebug         = 1
)

type HelmChartRepoClient struct {
	log logr.Logger
}

func NewHelmChartRepoClient(log logr.Logger, context *lsv1alpha1.Context) (*HelmChartRepoClient, error) {
	return &HelmChartRepoClient{
		log: log,
	}, nil
}

// fetchRepoCatalog returns the catalog of a helm chart repository
func (c *HelmChartRepoClient) fetchRepoCatalog(ctx context.Context, repoURL string) (*repo.IndexFile, error) {
	data, err := c.executeGetRequest(ctx, repoURL)
	if err != nil {
		return nil, err
	}

	index, sha := getCatalogCache().getCatalogFromCache(repoURL, data)
	if index == nil {
		// index not found in the cache, parse it
		index, err = getCatalogCache().parseCatalog(data)
		if err != nil {
			return nil, err
		}
		getCatalogCache().storeCatalogInCache(repoURL, index, sha)
	}
	return index, nil
}

// fetchChart returns the helm chart with the given URL
func (c *HelmChartRepoClient) fetchChart(ctx context.Context, chartURL string) ([]byte, error) {
	return c.executeGetRequest(ctx, chartURL)
}

func (c *HelmChartRepoClient) executeGetRequest(_ context.Context, rawURL string) ([]byte, error) {
	httpClient, err := c.getHttpClient()
	if err != nil {
		return nil, err
	}

	req, err := c.getRequest(rawURL)
	if err != nil {
		return nil, err
	}

	res, err := (httpClient).Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	data, err := c.readResponseBody(res)
	if err != nil {
		return nil, err
	}

	return data, err
}

func (c *HelmChartRepoClient) getHttpClient() (*http.Client, error) {
	// Require the SystemCertPool unless the env var is explicitly set.
	caCertPool, err := x509.SystemCertPool()
	if err != nil {
		if _, ok := os.LookupEnv("TILLER_PROXY_ALLOW_EMPTY_CERT_POOL"); !ok {
			return nil, errors.Wrap(err, "could not create system cert pool object")
		}
		caCertPool = x509.NewCertPool()
	}

	httpClient := &http.Client{
		Timeout: time.Second * defaultTimeoutSeconds,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{
				RootCAs:    caCertPool,
				MinVersion: tls.VersionTLS12,
			},
		},
	}

	return httpClient, nil
}

func (c *HelmChartRepoClient) getRequest(rawURL string) (*http.Request, error) {
	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return nil, fmt.Errorf("could not parse URL %s: %w", rawURL, err)
	}

	req, err := http.NewRequest("GET", parsedURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("could not create request object: %w", err)
	}

	req.Header["User-Agent"] = []string{"landscaper"}

	return req, nil
}

func (c *HelmChartRepoClient) readResponseBody(res *http.Response) ([]byte, error) {
	if res == nil {
		return nil, errors.New("response must not be nil")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		err := fmt.Errorf("request failed with status code %v", res.StatusCode)

		if c.log.V(logLevelDebug).Enabled() {
			body, bodyReadErr := ioutil.ReadAll(res.Body)
			if bodyReadErr != nil {
				c.log.Error(err, err.Error(), "response status code without body", res.StatusCode)
				return nil, err
			}

			c.log.Error(err, err.Error(), "response status code with body", res.StatusCode, "response body", string(body))
		}

		return nil, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response body: %w", err)
	}

	return body, nil
}
