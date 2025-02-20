package resources

import (
	"context"
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Mutator[K client.Object] interface {
	Empty() K
	Mutate(res K) error
	String() string
}

func CreateOrUpdateResource[K client.Object](ctx context.Context, clt client.Client, def Mutator[K]) error {
	res := def.Empty()
	_, err := controllerutil.CreateOrUpdate(ctx, clt, res, func() error {
		return def.Mutate(res)
	})
	if err != nil {
		return fmt.Errorf("failed to create or update %s: %w", def.String(), err)
	}
	return nil
}

func DeleteResource[K client.Object](ctx context.Context, clt client.Client, def Mutator[K]) error {
	res := def.Empty()
	if err := clt.Delete(ctx, res); client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to delete %s: %w", def.String(), err)
	}
	return nil
}
