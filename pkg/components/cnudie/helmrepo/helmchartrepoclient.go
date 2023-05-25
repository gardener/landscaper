package helmrepo

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/oauth2/google"
	"helm.sh/helm/v3/pkg/repo"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	lserrors "github.com/gardener/landscaper/apis/errors"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
)

const (
	defaultTimeoutSeconds = 180
	authHeaderDefaultKey  = "authHeader"
)

type HelmChartRepoClient struct {
	auths            []helmv1alpha1.Auth
	contextNamespace string
	lsClient         client.Client
}

func NewHelmChartRepoClient(context *lsv1alpha1.Context, lsClient client.Client) (*HelmChartRepoClient, lserrors.LsError) {
	auths := []helmv1alpha1.Auth{}

	if context != nil && context.Configurations != nil {
		if rawAuths, ok := context.Configurations[helmv1alpha1.HelmChartRepoCredentialsKey]; ok {
			repoCredentials := helmv1alpha1.HelmChartRepoCredentials{}
			err := yaml.Unmarshal(rawAuths.RawMessage, &repoCredentials)
			if err != nil {
				return nil, lserrors.NewWrappedError(err, "NewHelmChartRepoClient", "ParsingAuths", err.Error(), lsv1alpha1.ErrorConfigurationProblem)
			}

			auths = repoCredentials.Auths

			for i := range auths {
				auths[i].URL = normalizeUrl(auths[i].URL)
			}

			sort.Slice(auths, func(i, j int) bool {
				if len(auths[i].URL) == len(auths[j].URL) {
					return auths[i].URL < auths[j].URL
				}
				return len(auths[i].URL) > len(auths[j].URL)
			})
		}
	}

	contextNamespace := ""
	if context != nil {
		contextNamespace = context.Namespace
	}

	return &HelmChartRepoClient{
		auths:            auths,
		contextNamespace: contextNamespace,
		lsClient:         lsClient,
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

func (c *HelmChartRepoClient) executeGetRequest(ctx context.Context, rawURL string) ([]byte, error) {
	authData := c.getAuthData(rawURL)

	httpClient, err := c.getHttpClient(authData)
	if err != nil {
		return nil, err
	}

	req, err := c.getRequest(ctx, authData, rawURL)
	if err != nil {
		return nil, err
	}

	res, err := (httpClient).Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	data, err := c.readResponseBody(ctx, res)
	if err != nil {
		return nil, err
	}

	return data, err
}

func (c *HelmChartRepoClient) getHttpClient(authData *helmv1alpha1.Auth) (*http.Client, error) {
	// Require the SystemCertPool unless the env var is explicitly set.
	caCertPool, err := x509.SystemCertPool()
	if err != nil {
		if _, ok := os.LookupEnv("TILLER_PROXY_ALLOW_EMPTY_CERT_POOL"); !ok {
			return nil, errors.Wrap(err, "could not create system cert pool object")
		}
		caCertPool = x509.NewCertPool()
	}

	if authData != nil && authData.CustomCAData != "" {
		// Append our cert to the system pool
		if ok := caCertPool.AppendCertsFromPEM([]byte(authData.CustomCAData)); !ok {
			return nil, fmt.Errorf("failed to append customCA to system cert pool")
		}
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

func (c *HelmChartRepoClient) getRequest(ctx context.Context, authData *helmv1alpha1.Auth, rawURL string) (*http.Request, error) {
	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return nil, fmt.Errorf("could not parse URL %s: %w", rawURL, err)
	}

	req, err := http.NewRequest("GET", parsedURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("could not create request object: %w", err)
	}

	req.Header["User-Agent"] = []string{"landscaper"}

	err = c.setAuthHeader(ctx, authData, req)
	if err != nil {
		return nil, lserrors.NewWrappedError(err, "getRequest", "setAuthHeader", "could not set auth header")
	}

	return req, nil
}

func (c *HelmChartRepoClient) setAuthHeader(ctx context.Context, authData *helmv1alpha1.Auth, req *http.Request) error {
	if authData == nil {
		return nil
	}

	authHeader, err := c.getAuthHeader(ctx, authData)
	if err != nil {
		return err
	}

	if strings.HasPrefix(authHeader, "Basic ") {
		trimmedBasicHeader := strings.TrimPrefix(authHeader, "Basic ")
		username, password, err := c.decodeBasicAuthCredentials(trimmedBasicHeader)
		if err != nil {
			return err
		}
		if username == "_json_key" {
			accessToken, err := c.getGCloudAccessToken(password)
			if err != nil {
				return err
			}
			authHeader = "Bearer " + accessToken
		}
	}

	req.Header.Set("Authorization", authHeader)
	return nil
}

func (c *HelmChartRepoClient) getAuthHeader(ctx context.Context, authData *helmv1alpha1.Auth) (string, error) {
	if len(authData.AuthHeader) > 0 && authData.SecretRef != nil {
		return "", fmt.Errorf("failed to get auth header: auth header and secret ref are both set")
	}

	if len(authData.AuthHeader) > 0 {
		return authData.AuthHeader, nil
	}

	if authData.SecretRef != nil {
		secretKey := client.ObjectKey{Name: authData.SecretRef.Name, Namespace: c.contextNamespace}
		secret := &corev1.Secret{}
		if err := c.lsClient.Get(ctx, secretKey, secret); err != nil {
			return "", err
		}

		authHeaderKey := authData.SecretRef.Key
		if len(authData.SecretRef.Key) == 0 {
			authHeaderKey = authHeaderDefaultKey
		}

		authHeader, ok := secret.Data[authHeaderKey]
		if !ok {
			return "", fmt.Errorf("failed to get auth header: key %s not found in secret", authHeaderKey)
		}

		return string(authHeader), nil
	}

	return "", fmt.Errorf("failed to get auth header: neither auth header nor secret ref is set")
}

func (c *HelmChartRepoClient) decodeBasicAuthCredentials(base64EncodedBasicAuthCredentials string) (string, string, lserrors.LsError) {
	decodedCredentials, err := base64.StdEncoding.DecodeString(base64EncodedBasicAuthCredentials)
	if err != nil {
		return "", "", lserrors.NewWrappedError(err, "decodeBasicAuthCredentials", "DecodeString",
			"Couldn't decode basic auth credentials", lsv1alpha1.ErrorConfigurationProblem)
	}
	splittedCredentials := strings.SplitN(string(decodedCredentials), ":", 2)
	if len(splittedCredentials) < 2 {
		return "", "", lserrors.NewError("decodeBasicAuthCredentials", "SplitN",
			"Password missing in credential string. Could not split by colon ':'", lsv1alpha1.ErrorConfigurationProblem)
	}

	username := splittedCredentials[0]
	password := splittedCredentials[1]
	return username, password, nil
}

func (c *HelmChartRepoClient) getGCloudAccessToken(gcloudServiceAccountJSON string) (string, error) {
	jwtConfig, err := google.JWTConfigFromJSON([]byte(gcloudServiceAccountJSON), "https://www.googleapis.com/auth/devstorage.read_only")
	if err != nil {
		return "", errors.Wrap(err, "Couldn't create Google Service Account object")
	}
	tokenSource := jwtConfig.TokenSource(context.TODO())
	token, err := tokenSource.Token()
	if err != nil {
		return "", errors.Wrap(err, "Couldn't fetch token from token source")
	}
	return token.AccessToken, nil
}

func (c *HelmChartRepoClient) getAuthData(rawURL string) *helmv1alpha1.Auth {
	for _, auth := range c.auths {
		if strings.HasPrefix(rawURL, auth.URL) {
			return &auth
		}
	}

	return nil
}

func (c *HelmChartRepoClient) readResponseBody(ctx context.Context, res *http.Response) ([]byte, error) {
	logger, _ := logging.FromContextOrNew(ctx, []interface{}{lc.KeyMethod, "readResponseBody"})

	if res == nil {
		return nil, errors.New("response must not be nil")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		err := fmt.Errorf("request failed with status code %v", res.StatusCode)

		if logger.Enabled(logging.DEBUG) {
			body, bodyReadErr := io.ReadAll(res.Body)
			if bodyReadErr != nil {
				logger.Error(err, err.Error(), "response status code without body", res.StatusCode)
				return nil, err
			}

			logger.Error(err, err.Error(), "response status code with body", res.StatusCode, "response body", string(body))
		}

		return nil, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response body: %w", err)
	}

	return body, nil
}
