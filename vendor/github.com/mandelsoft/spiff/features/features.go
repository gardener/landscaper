package features

import (
	"fmt"
	"os"
	"strings"
)

const INTERPOLATION = "interpolation"
const CONTROL = "control"

type FeatureFlags map[string]struct{}

func (this FeatureFlags) Enabled(name string) bool {
	if this == nil {
		return false
	}
	_, ok := this[name]
	return ok
}

func (this FeatureFlags) Set(name string, active bool) error {
	no := strings.HasPrefix(name, "no")
	if no {
		name = name[2:]
	}
	switch name {
	case INTERPOLATION:
	case CONTROL:
	default:
		return fmt.Errorf("unknown feature flag %q", name)
	}
	if active != no {
		this[name] = struct{}{}
	} else {
		delete(this, name)
	}
	return nil
}

func (this FeatureFlags) Size() int {
	return len(this)
}

func (this FeatureFlags) InterpolationEnabled() bool {
	return this.Enabled(INTERPOLATION)
}
func (this FeatureFlags) SetInterpolation(active bool) {
	this.Set(INTERPOLATION, active)
}

func (this FeatureFlags) ControlEnabled() bool {
	return this.Enabled(CONTROL)
}
func (this FeatureFlags) SetControl(active bool) {
	this.Set(CONTROL, active)
}

func Features() FeatureFlags {
	features := FeatureFlags{}
	// setup defaults
	setting := os.Getenv("SPIFF_FEATURES")
	for _, f := range strings.Split(setting, ",") {
		features.Set(strings.TrimSpace(f), true)
	}
	return features
}

func EncryptionKey() string {
	return os.Getenv("SPIFF_ENCRYPTION_KEY")
}
