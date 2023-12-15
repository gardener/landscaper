package matchers

import (
	"context"
	"errors"
	"fmt"

	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type existMatcher struct {
	client client.Client
}

// Exist succeeds if actual is an existing resource.
func Exist(client client.Client) types.GomegaMatcher {
	return &existMatcher{
		client: client,
	}
}

func ExistForClient(client client.Client) func() types.GomegaMatcher {
	return func() types.GomegaMatcher {
		return Exist(client)
	}
}

func (m *existMatcher) Match(actual interface{}) (success bool, err error) {
	obj, ok := actual.(client.Object)
	if !ok {
		return false, errors.New(format.Message(actual, "to be a client.Object"))
	}

	err = m.client.Get(context.Background(), client.ObjectKeyFromObject(obj), obj)
	if err == nil {
		return true, nil
	} else if apierrors.IsNotFound(err) {
		return false, nil
	} else {
		return false, err
	}
}

func (m *existMatcher) FailureMessage(actual interface{}) (message string) {
	obj, ok := actual.(client.Object)
	if !ok {
		format.Message(actual, "to be a client.Object")
	}

	key := fmt.Sprintf("%s %s", obj.GetObjectKind().GroupVersionKind().Kind, client.ObjectKeyFromObject(obj).String())
	return format.Message(key, "to exist")
}

func (m *existMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	obj, ok := actual.(client.Object)
	if !ok {
		format.Message(actual, "to be a client.Object")
	}

	key := fmt.Sprintf("%s %s", obj.GetObjectKind().GroupVersionKind().Kind, client.ObjectKeyFromObject(obj).String())
	return format.Message(key, "not to exist")
}
