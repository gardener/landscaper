package utils

import (
	"sync"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
)

type WorkerCounter struct {
	maxNumOfWorker int
	counter        int
	mutex          sync.RWMutex
}

func NewWorkerCounter(maxNumOfWorker int) *WorkerCounter {
	return &WorkerCounter{
		maxNumOfWorker: maxNumOfWorker,
	}
}

func (r *WorkerCounter) Enter() int {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.counter++

	result := r.counter
	return result
}

func (r *WorkerCounter) EnterWithLog(log logging.Logger, percentageBorder int) int {
	result := r.Enter()

	if result*100 >= r.maxNumOfWorker*percentageBorder {
		log.Info("worker threads of controller", "usedWorkerThreads", result)
	}

	return result
}

func (r *WorkerCounter) Exit() int {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.counter--

	result := r.counter
	return result
}
