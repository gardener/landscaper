package lock

import "github.com/gardener/landscaper/apis/config"

func IsLockingEnabledForMainControllers(config *config.LandscaperConfiguration) bool {
	return config != nil &&
		config.HPAMainConfiguration != nil &&
		config.HPAMainConfiguration.MaxReplicas > 1
}
