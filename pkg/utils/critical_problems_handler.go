package utils

import (
	"context"

	"github.com/gardener/landscaper/apis/core/v1alpha1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
)

const objectName = "critical-problems"

type CriticalProblemsHandler interface {
	ReportProblem(ctx context.Context, hostUncachedClient client.Client, description string)
	GetCriticalProblems(ctx context.Context, hostUncachedClient client.Client) (*v1alpha1.CriticalProblems, error)
	AccessAllowed(ctx context.Context, hostUncachedClient client.Client) error
}

var cph CriticalProblemsHandler = &criticalProblemsHandler{}

func GetCriticalProblemsHandler() CriticalProblemsHandler {
	return cph
}

type criticalProblemsHandler struct {
}

func (r *criticalProblemsHandler) ReportProblem(ctx context.Context, hostUncachedClient client.Client, description string) {
	logger, _ := logging.FromContextOrNew(ctx, nil)

	problems := r.getEmptyCriticalProblems()

	if err := hostUncachedClient.Get(ctx, client.ObjectKeyFromObject(problems), problems); err != nil {
		if apierrors.IsNotFound(err) {
			problems.Spec.CriticalProblems = []v1alpha1.CriticalProblem{
				{
					PodName:      GetCurrentPodNamespace(),
					CreationTime: metav1.Now(),
					Description:  description,
				},
			}

			if err = hostUncachedClient.Create(ctx, problems); err != nil {
				logger.Error(err, "ReportProblem during creation")
			}
		} else {
			logger.Error(err, "ReportProblem fetching object")
		}

		return
	}

	problem := v1alpha1.CriticalProblem{
		PodName:      GetCurrentPodNamespace(),
		CreationTime: metav1.Now(),
		Description:  description,
	}
	problems.Spec.CriticalProblems = append(problems.Spec.CriticalProblems, problem)

	startIndex := len(problems.Spec.CriticalProblems) - 10
	if startIndex < 0 {
		startIndex = 0
	}

	problems.Spec.CriticalProblems = problems.Spec.CriticalProblems[startIndex:]

	if err := hostUncachedClient.Update(ctx, problems); err != nil {
		logger.Error(err, "ReportProblem updating object")
	}
}

func (r *criticalProblemsHandler) GetCriticalProblems(ctx context.Context,
	hostUncachedClient client.Client) (*v1alpha1.CriticalProblems, error) {

	problems := r.getEmptyCriticalProblems()
	err := hostUncachedClient.Get(ctx, client.ObjectKeyFromObject(problems), problems)
	return problems, err
}

func (r *criticalProblemsHandler) AccessAllowed(ctx context.Context,
	hostUncachedClient client.Client) error {
	_, err := r.GetCriticalProblems(ctx, hostUncachedClient)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

func (r *criticalProblemsHandler) getEmptyCriticalProblems() *v1alpha1.CriticalProblems {
	problems := &v1alpha1.CriticalProblems{}
	problems.SetName(objectName)
	problems.SetNamespace(GetCurrentPodNamespace())
	return problems
}
