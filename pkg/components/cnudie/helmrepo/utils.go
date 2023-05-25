package helmrepo

import "strings"

func normalizeUrl(url string) string {
	result := strings.TrimSpace(url)
	result = strings.TrimSuffix(result, "/")
	result = strings.TrimPrefix(result, "https://")
	return "https://" + result
}
