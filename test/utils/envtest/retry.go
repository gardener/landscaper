package envtest

import (
	"context"
	"github.com/gardener/landscaper/hack/testcluster/pkg/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

func GetWithRetry(ctx context.Context, c client.Client, log utils.Logger, key client.ObjectKey, obj client.Object,
	opts ...client.GetOption) error {
	if retryError := RetrySporadic(log, func() error {
		return c.Get(ctx, key, obj, opts...)
	}); retryError != nil {
		if log != nil {
			log.Logln("state CreateWithClient-create failed: " + retryError.Error())
		}
		return retryError
	}

	return nil
}

func RetrySporadic(log utils.Logger, fn func() error) error {
	retries := 10

	for i := 0; i < retries; i++ {
		err := fn()
		if err == nil {
			return nil
		} else if i == retries-1 {
			if log != nil {
				log.Logfln("after %d attempts, the last retry still failed: %w", retries, err)
			}
			return err
		} else if !checkIfSporadicError(err) {
			if log != nil {
				log.Logfln("stop retrying after non-sporadic error: %w", err)
			}
			return err
		} else {
			if log != nil {
				log.Logfln("continue retrying after sporadic error: %w", err)
			}
			time.Sleep(5 * time.Second)
		}
	}

	return nil
}

func checkIfSporadicError(err error) bool {
	return strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "context deadline exceeded") ||
		strings.Contains(err.Error(), "failed to call webhook")
}
