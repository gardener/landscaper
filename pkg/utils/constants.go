package utils

import (
	"fmt"
	"os"
	"reflect"
	"strconv"

	"k8s.io/client-go/rest"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
)

const (
	NoPodname      = "noPodname"
	NoPodnamespace = "noPodnamespace"

	LsHostClientBurstVar     = "LS_HOST_CLIENT_BURST"
	LsHostClientQpsVar       = "LS_HOST_CLIENT_QPS"
	LsResourceClientBurstVar = "LS_RESOURCE_CLIENT_BURST"
	LsResourceClientQpsVar   = "LS_RESOURCE_CLIENT_QPS"

	LsHostClientBurstDefault     = 30
	LsHostClientQpsDefault       = 20
	LsResourceClientBurstDefault = 60
	LsResourceClientQpsDefault   = 40
)

func GetCurrentPodName() string {
	result := os.Getenv("MY_POD_NAME")

	if result == "" {
		result = NoPodname
	}

	return result
}

func GetCurrentPodNamespace() string {
	result := os.Getenv("MY_POD_NAMESPACE")

	if result == "" {
		result = NoPodnamespace
	}

	return result
}

func IsDeployItemPhase(di *lsv1alpha1.DeployItem, phase lsv1alpha1.DeployItemPhase) bool {
	return di.Status.Phase == phase
}

func IsInstallationPhase(inst *lsv1alpha1.Installation, phase lsv1alpha1.InstallationPhase) bool {
	return inst.Status.InstallationPhase == phase
}

func IsDeployItemJobIDsIdentical(di *lsv1alpha1.DeployItem) bool {
	return di.Status.GetJobID() == di.Status.JobIDFinished
}

func IsInstallationJobIDsIdentical(inst *lsv1alpha1.Installation) bool {
	return inst.Status.JobID == inst.Status.JobIDFinished
}

func IsExecutionJobIDsIdentical(exec *lsv1alpha1.Execution) bool {
	return exec.Status.JobID == exec.Status.JobIDFinished
}

func RestConfigWithModifiedClientRequestRestrictions(log logging.Logger, restConfig *rest.Config, burst, qps int) *rest.Config {
	modifiedRestConfig := *restConfig

	if restConfig.RateLimiter != nil {
		log.Info("ClientRequestRestrictions - RateLimiter: " + reflect.TypeOf(restConfig.RateLimiter).String())
	}
	log.Info("ClientRequestRestrictions - OldBurst: " + strconv.Itoa(restConfig.Burst))
	log.Info("ClientRequestRestrictions - OldQPS: " + fmt.Sprintf("%v", restConfig.QPS))

	modifiedRestConfig.RateLimiter = nil
	modifiedRestConfig.Burst = burst
	modifiedRestConfig.QPS = float32(qps)

	return &modifiedRestConfig
}

func GetHostClientRequestRestrictions(log logging.Logger, hostAndResourceClusterDifferent bool) (int, int) {
	burst, qps := GetResourceClientRequestRestrictions(log)

	if hostAndResourceClusterDifferent {
		burst = getClientRestriction(LsHostClientBurstVar, LsHostClientBurstDefault)
		qps = getClientRestriction(LsHostClientQpsVar, LsHostClientQpsDefault)
	}

	log.Info("HostClientRequestRestrictions", LsHostClientBurstVar, burst, LsHostClientQpsVar, qps)

	return burst, qps
}

func GetResourceClientRequestRestrictions(log logging.Logger) (int, int) {
	burst := getClientRestriction(LsResourceClientBurstVar, LsResourceClientBurstDefault)
	qps := getClientRestriction(LsResourceClientQpsVar, LsResourceClientQpsDefault)

	log.Info("ResourceClientRequestRestrictions", LsResourceClientBurstVar, burst, LsResourceClientQpsVar, qps)
	return burst, qps
}

func getClientRestriction(envName string, defaultVal int) int {
	tmpStr := os.Getenv(envName)

	if len(tmpStr) > 0 {
		tmpInt, err := strconv.Atoi(tmpStr)
		if err == nil && tmpInt > 0 {
			return tmpInt
		}
	}

	return defaultVal
}
