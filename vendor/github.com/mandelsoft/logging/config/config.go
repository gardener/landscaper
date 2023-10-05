/*
 * Copyright 2022 Mandelsoft. All rights reserved.
 *  This file is licensed under the Apache Software License, v. 2 except as noted
 *  otherwise in the LICENSE file
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package config

import (
	"sigs.k8s.io/yaml"

	"github.com/mandelsoft/logging"
)

type Config struct {
	DefaultLevel string `json:"defaultLevel,omitempty"`
	Rules        []Rule `json:"rules,omitempty"`
}

func (c *Config) UnmarshalFrom(data []byte) error {
	return yaml.Unmarshal(data, c)
}

func EvaluateFromData(data []byte) (*Config, error) {
	return _registry.EvaluateFromData(data)
}

func Evaluate(cfg *Config) error {
	return _registry.Evaluate(cfg)
}

func ConfigureWithData(ctx logging.Context, data []byte) error {
	return _registry.ConfigureWithData(ctx, data)
}

func Configure(ctx logging.Context, cfg *Config) error {
	return _registry.Configure(ctx, cfg)
}
