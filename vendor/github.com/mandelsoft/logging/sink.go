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
	"github.com/go-logr/logr"
)

type sink struct {
	level int
	delta int
	sink  logr.LogSink
}

var _ logr.LogSink = (*sink)(nil)

func WrapSink(level, delta int, orig logr.LogSink) logr.LogSink {
	return &sink{
		level: level,
		delta: delta,
		sink:  orig,
	}
}

func (s *sink) Init(info logr.RuntimeInfo) {
	s.sink.Init(info)
}

func (s *sink) Enabled(level int) bool {
	// leave the final (non-local) decision to the underlying sink
	// to offer the possibility to disable local error logging
	// independent of the underlying sink, to stick to the logr
	// error contract (avoiding the sink level to disable error log).
	return s.level >= level // && s.sink.Enabled(level+s.delta)
}

func (s *sink) Info(level int, msg string, keysAndValues ...interface{}) {
	if !s.Enabled(level) {
		return
	}
	s.sink.Info(level+s.delta, msg, keysAndValues...)
}

func (s sink) Error(err error, msg string, keysAndValues ...interface{}) {
	s.sink.Error(err, msg, keysAndValues...)
}

func (s *sink) WithValues(keysAndValues ...interface{}) logr.LogSink {
	return &sink{
		level: s.level,
		delta: s.delta,
		sink:  s.sink.WithValues(keysAndValues...),
	}
}

func (s sink) WithName(name string) logr.LogSink {
	return &sink{
		level: s.level,
		delta: s.delta,
		sink:  s.sink.WithName(name),
	}
}

////////////////////////////////////////////////////////////////////////////////

func AsLevelFunc(lvl int) LevelFunc {
	return func() int {
		return lvl
	}
}

func AsSinkFunc(s logr.LogSink) SinkFunc {
	return func() logr.LogSink {
		return s
	}
}

type dynsink struct {
	level LevelFunc
	delta int
	sink  SinkFunc
}

var _ logr.LogSink = (*sink)(nil)

func DynSink(level LevelFunc, delta int, orig SinkFunc) logr.LogSink {
	return &dynsink{
		level: level,
		delta: delta,
		sink:  orig,
	}
}

func (s *dynsink) Init(info logr.RuntimeInfo) {
	s.sink().Init(info)
}

func (s *dynsink) Enabled(level int) bool {
	// leave the final (non-local) decision to the underlying sink
	// to offer the possibility to disable local error logging
	// independent of the underlying sink, to stick to the logr
	// error contract (avoiding the sink level to disable error log).
	return s.level() >= level // && s.sink.Enabled(level+s.delta)
}

func (s *dynsink) Info(level int, msg string, keysAndValues ...interface{}) {
	if !s.Enabled(level) {
		return
	}
	s.sink().Info(level+s.delta, msg, keysAndValues...)
}

func (s dynsink) Error(err error, msg string, keysAndValues ...interface{}) {
	s.sink().Error(err, msg, keysAndValues...)
}

func (s *dynsink) WithValues(keysAndValues ...interface{}) logr.LogSink {
	return &dynsink{
		level: s.level,
		delta: s.delta,
		sink:  func() logr.LogSink { return s.sink().WithValues(keysAndValues...) },
	}
}

func (s dynsink) WithName(name string) logr.LogSink {
	return &dynsink{
		level: s.level,
		delta: s.delta,
		sink:  func() logr.LogSink { return s.sink().WithName(name) },
	}
}
