// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"sort"
	"strings"
	"sync"
)

type AppliedConfigSelector interface {
	Select(cfg *AppliedConfig) bool
}

type AppliedConfigSelectorFunction func(cfg *AppliedConfig) bool

func (f AppliedConfigSelectorFunction) Select(cfg *AppliedConfig) bool { return f(cfg) }

var AllAppliedConfigs = AppliedConfigSelectorFunction(func(*AppliedConfig) bool { return true })

func AppliedGenerationSelector(gen int64) AppliedConfigSelector {
	return AppliedConfigSelectorFunction(func(cfg *AppliedConfig) bool {
		return cfg.generation > gen
	})
}

func AppliedVersionSelector(v string) AppliedConfigSelector {
	return AppliedConfigSelectorFunction(func(cfg *AppliedConfig) bool {
		return cfg.config.GetVersion() == v
	})
}

func AppliedConfigSelectorFor(s ConfigSelector) AppliedConfigSelector {
	if s == nil {
		return nil
	}
	return AppliedConfigSelectorFunction(func(cfg *AppliedConfig) bool {
		return s.Select(cfg.config)
	})
}

func AppliedAndSelector(and ...AppliedConfigSelector) AppliedConfigSelector {
	return AppliedConfigSelectorFunction(func(cfg *AppliedConfig) bool {
		for _, a := range and {
			if a != nil && !a.Select(cfg) {
				return false
			}
		}
		return true
	})
}

type AppliedConfigs []*AppliedConfig

func (l AppliedConfigs) Len() int           { return len(l) }
func (l AppliedConfigs) Less(i, j int) bool { return l[i].generation < l[j].generation }
func (l AppliedConfigs) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }

func (l AppliedConfigs) Configs() []Config {
	n := make([]Config, len(l))
	for i, e := range l {
		n[i] = e.config
	}
	return n
}

type AppliedConfig struct {
	generation  int64
	config      Config
	description string
}

func (c *AppliedConfig) eval(ctx Context) Config {
	if e, ok := c.config.(Evaluator); ok {
		n, err := e.Evaluate(ctx)
		if err == nil {
			c.config = n
		}
	}
	return c.config
}

type ConfigStore struct {
	lock       sync.RWMutex
	generation int64
	types      map[string]AppliedConfigs
	configs    AppliedConfigs
}

func NewConfigStore() *ConfigStore {
	return &ConfigStore{
		types: map[string]AppliedConfigs{},
	}
}

func (s *ConfigStore) Generation() int64 {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.generation
}

func (s *ConfigStore) Reset() int64 {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.configs = nil
	s.types = map[string]AppliedConfigs{}
	return s.generation
}

func (s *ConfigStore) Apply(c Config, desc string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.generation++
	a := &AppliedConfig{s.generation, c, desc}
	configs := s.types[c.GetKind()]
	s.types[c.GetKind()] = append(configs, a)
	s.configs = append(s.configs, a)
}

func (s *ConfigStore) appendCfg(ctx Context, result, configs AppliedConfigs, selector AppliedConfigSelector) AppliedConfigs {
	if selector == nil {
		selector = AllAppliedConfigs
	}
	for i, a := range configs {
		a.eval(ctx)
		if selector.Select(a) {
			configs[i] = a
			result = append(result, a)
		}
	}
	return result
}

func (c *ConfigStore) GetConfigForSelector(ctx Context, selector AppliedConfigSelector) (int64, AppliedConfigs) {
	var result AppliedConfigs
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.generation, c.appendCfg(ctx, result, c.configs, selector)
}

func (c *ConfigStore) GetConfigForName(ctx Context, name string, selector AppliedConfigSelector) (int64, AppliedConfigs) {
	var result AppliedConfigs
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.generation, c.appendCfg(ctx, result, c.types[name], selector)
}

func (c *ConfigStore) GetConfigForType(ctx Context, typ string, selector AppliedConfigSelector) (int64, AppliedConfigs) {
	var result AppliedConfigs
	c.lock.Lock()
	defer c.lock.Unlock()

	result = c.appendCfg(ctx, result, c.types[typ], selector)
	idx := strings.LastIndex(typ, "/")
	if idx > 0 {
		name := typ[:idx]
		version := typ[idx:]
		result = c.appendCfg(ctx, result, c.types[name], AppliedAndSelector(AppliedVersionSelector(version), selector))
	}
	sort.Sort(result)
	return c.generation, result
}
