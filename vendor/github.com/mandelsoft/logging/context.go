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

package logging

import (
	"sync"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/logging/logrusl/adapter"
	"github.com/mandelsoft/logging/logrusr"
)

type context struct {
	id      ContextId
	lock    sync.RWMutex
	base    Context
	updater *Updater

	level int
	sink  logr.LogSink
	rules []Rule

	defaultLogger Logger

	messageContext []MessageContext
	effLevel       int
	effSink        logr.LogSink
}

var _ Context = (*context)(nil)
var _ ContextProvider = (*context)(nil)

func NewDefault() Context {
	return NewWithBase(nil)
}

func New(logger logr.Logger) Context {
	return NewWithBase(nil, logger)
}

func NewWithBase(base Context, baselogger ...logr.Logger) Context {
	return newWithBase(base, baselogger...)
}

func newWithBase(base Context, baselogger ...logr.Logger) *context {
	ctx := &context{
		base:  base,
		level: -1,
		id:    getId(),
	}

	if base == nil {
		ctx.level = InfoLevel
		ctx.updater = NewUpdater(nil)
	} else {
		internal := base.Tree()
		ctx.updater = NewUpdater(internal.Updater())
		ctx.messageContext = internal.GetMessageContext()
	}

	if len(baselogger) > 0 {
		ctx.setBaseLogger(baselogger[0])
	}
	if base == nil && len(baselogger) == 0 {
		l := adapter.NewLogger()
		l.Formatter = adapter.NewTextFmtFormatter()
		ctx.setBaseLogger(logrusr.New(l))
	}
	ctx.defaultLogger = NewLogger(DynSink(ctx.GetDefaultLevel, 0, ctx.GetSink))
	ctx._update()
	return ctx
}

func (c *context) Tree() ContextSupport {
	return c
}

func (c *context) GetMessageContext() []MessageContext {
	return sliceCopy(c.messageContext)
}

func (c *context) Updater() *Updater {
	return c.updater
}

func (c *context) update() {
	if !c.updater.Require() {
		return
	}
	c._update()
}

func (c *context) _update() {
	if c.level < 0 {
		c.effLevel = c.base.GetDefaultLevel()
	} else {
		c.effLevel = c.level
	}

	if c.sink == nil {
		c.effSink = c.base.GetSink()
	} else {
		c.effSink = c.sink
	}
}

func (c *context) LoggingContext() Context {
	return c
}

func (c *context) GetSink() logr.LogSink {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.update()
	return c.effSink
}

func (c *context) GetBaseContext() Context {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.base
}

func (c *context) GetDefaultLogger() Logger {
	return c.defaultLogger
}

func (c *context) GetDefaultLevel() int {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.update()
	return c.effLevel
}

func (c *context) SetDefaultLevel(level int) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.setDefaultLevel(level)
	c.updater.Modify()
	c._update()
}

func (c *context) setDefaultLevel(level int) {
	c.level = level
}

func (c *context) SetBaseLogger(logger logr.Logger, plain ...bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.setBaseLogger(logger, plain...)
	c.updater.Modify()
	c._update()
}

func (c *context) setBaseLogger(logger logr.Logger, plain ...bool) {
	if len(plain) == 0 || !plain[0] {
		c.sink = shifted(logger)
	} else {
		c.sink = logger.GetSink()
	}
}

func (c *context) AddRule(rules ...Rule) {
	c.lock.Lock()
	defer c.lock.Unlock()

	for _, rule := range rules {
		if rule != nil {
			if upd, ok := rule.(UpdatableRule); ok {
				i := 0
				for i < len(c.rules) {
					f := c.rules[i]
					if upd.MatchRule(f) {
						c.rules = append(c.rules[:i], c.rules[i+1:]...)
					} else {
						i++
					}
				}
			}
			c.rules = append(append(c.rules[:0:0], rule), c.rules...)
		}
	}
	c.updater.Modify()
}

func (c *context) AddRulesTo(ctx Context) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	rules := make([]Rule, len(c.rules))
	for i := range rules {
		rules[i] = c.rules[len(c.rules)-i-1]
	}
	ctx.AddRule(rules...)
}

func (c *context) ResetRules() {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.rules = nil
	c.updater.Modify()
}

func (c *context) WithContext(messageContext ...MessageContext) Context {
	ctx := newWithBase(c)
	ctx.messageContext = JoinMessageContext(ctx.messageContext, messageContext...)
	return ctx
}

func (c *context) Logger(messageContext ...MessageContext) Logger {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if len(c.messageContext) > 0 {
		messageContext = JoinMessageContext(c.messageContext, messageContext...)
	}
	l := c.evaluate(c.GetSink, messageContext...)
	if l == nil {
		l = c.defaultLogger
	}
	for _, c := range messageContext {
		if a, ok := c.(Attacher); ok {
			l = a.Attach(l)
		}
	}
	return l
}

func (c *context) V(level int, messageContext ...MessageContext) logr.Logger {
	return c.Logger(messageContext...).V(level)
}

func (c *context) Evaluate(base SinkFunc, messageContext ...MessageContext) Logger {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.evaluate(base, messageContext...)
}

func (c *context) evaluate(base SinkFunc, messageContext ...MessageContext) Logger {
	for _, rule := range c.rules {
		l := rule.Match(base, messageContext...)
		if l != nil {
			return l
		}
	}
	if c.base != nil {
		return c.base.Evaluate(base, messageContext...)
	}
	return nil
}

type levelCatcher struct {
	sink
}

func (h *levelCatcher) Enabled(l int) bool {
	h.level = l
	return false
}

func getLogrLevel(l logr.Logger) int {
	var s levelCatcher
	l.WithSink(&s).Enabled()
	return s.level
}

func shifted(logger logr.Logger) logr.LogSink {
	level := getLogrLevel(logger)
	return WrapSink(level+9, level, logger.GetSink())
}

func JoinMessageContext(base []MessageContext, list ...MessageContext) []MessageContext {
	return sliceAppend(base, list...)
}
