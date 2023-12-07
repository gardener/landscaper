package envtest

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/hack/testcluster/pkg/utils"
)

type retryingClient struct {
	client.Client
	log utils.Logger
}

func NewRetryingClient(innerClient client.Client, log utils.Logger) client.Client {
	if log == nil {
		log = utils.NewDiscardLogger()
	}

	return &retryingClient{
		Client: innerClient,
		log:    log,
	}
}

func (r *retryingClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	err := retrySporadic(ctx, r.log, func() error {
		return r.Client.Get(ctx, key, obj, opts...)
	})

	if err != nil {
		err = errors.Wrap(err, "retryingClient: "+err.Error())
	}

	return err
}

func (r *retryingClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	err := retrySporadic(ctx, r.log, func() error {
		return r.Client.List(ctx, list, opts...)
	})

	if err != nil {
		err = errors.Wrap(err, "retryingClient: "+err.Error())
	}

	return err
}

func (r *retryingClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	err := retrySporadic(ctx, r.log, func() error {
		return r.Client.Create(ctx, obj, opts...)
	})

	if err != nil {
		err = errors.Wrap(err, "retryingClient: "+err.Error())
	}

	return err
}

func (r *retryingClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	err := retrySporadic(ctx, r.log, func() error {
		return r.Client.Update(ctx, obj, opts...)
	})

	if err != nil {
		err = errors.Wrap(err, "retryingClient: "+err.Error())
	}

	return err
}

func (r *retryingClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	err := retrySporadic(ctx, r.log, func() error {
		return r.Client.Patch(ctx, obj, patch, opts...)
	})

	if err != nil {
		err = errors.Wrap(err, "retryingClient: "+err.Error())
	}

	return err
}

func (r *retryingClient) Status() client.SubResourceWriter {
	return &retryingSubResourceWriter{
		SubResourceWriter: r.Client.Status(),
		log:               r.log,
	}
}

type retryingSubResourceWriter struct {
	client.SubResourceWriter
	log utils.Logger
}

func (r *retryingSubResourceWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	err := retrySporadic(ctx, r.log, func() error {
		return r.SubResourceWriter.Update(ctx, obj, opts...)
	})

	if err != nil {
		err = errors.Wrap(err, "retryingSubResourceWriter: "+err.Error())
	}

	return err
}

func retrySporadic(ctx context.Context, log utils.Logger, fn func() error) error {
	retries := 10

	for i := 0; i < retries; i++ {
		if err := ctx.Err(); err != nil {
			log.Logfln("retrying client: context is closed: %w", err)
			return err
		}

		err := fn()
		if err == nil {
			return nil
		} else if i == retries-1 {
			log.Logfln("retrying client: all attempts failed: %w", err)
			return err
		} else if !isSporadicError(err) {
			log.Logfln("retrying client: stop retrying after non-sporadic error: %w", err)
			return err
		} else if ctxError := ctx.Err(); ctxError != nil {
			log.Logfln("retrying client: stop retrying because context is closed: %w", ctxError)
			return ctxError
		} else {
			log.Logfln("retrying client: continue retrying after sporadic error: %w", err)
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
