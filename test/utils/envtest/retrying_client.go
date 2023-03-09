package envtest

import (
	"context"
	"strings"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/hack/testcluster/pkg/utils"
)

type RetryingClient struct {
	client.Client
	log utils.Logger
}

func NewRetryingClient(innerClient client.Client, log utils.Logger) client.Client {
	if log == nil {
		log = utils.NewDiscardLogger()
	}

	return &RetryingClient{
		Client: innerClient,
		log:    log,
	}
}

func (r *RetryingClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	return r.retrySporadic(func() error {
		return r.Client.Get(ctx, key, obj, opts...)
	})
}

func (r *RetryingClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return r.retrySporadic(func() error {
		return r.Client.List(ctx, list, opts...)
	})
}

func (r *RetryingClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	return r.retrySporadic(func() error {
		return r.Client.Create(ctx, obj, opts...)
	})
}

func (r *RetryingClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return r.retrySporadic(func() error {
		return r.Client.Update(ctx, obj, opts...)
	})
}

func (r *RetryingClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return r.retrySporadic(func() error {
		return r.Client.Patch(ctx, obj, patch, opts...)
	})
}

func (r *RetryingClient) retrySporadic(fn func() error) error {
	retries := 10

	for i := 0; i < retries; i++ {
		err := fn()
		if err == nil {
			return nil
		} else if i == retries-1 {
			r.log.Logfln("retrying client: all attempts failed: %w", err)
			return err
		} else if !isSporadicError(err) {
			return err
		} else {
			r.log.Logfln("retrying client: continue retrying after sporadic error: %w", err)
			time.Sleep(3 * time.Second)
		}
	}

	return nil
}

func isSporadicError(err error) bool {
	return strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "context deadline exceeded") ||
		strings.Contains(err.Error(), "failed to call webhook") ||
		strings.Contains(err.Error(), "connection reset by peer")
}
