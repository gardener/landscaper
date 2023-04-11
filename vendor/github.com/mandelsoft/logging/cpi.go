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

// ContextSupport is intended for Context implementations for working
// together in a context tree consiting of potentially different implementations.
// It is not intended for the consumer of a logging context.
type ContextSupport interface {
	// UpdateState provides information of the update watermark in a context tree
	Updater() *Updater

	// GetBaseContext returns the base context for nested logging contexts.
	GetBaseContext() Context

	// GetMessageContext returns the configured standard message context
	// shared for all created Loggers.
	GetMessageContext() []MessageContext
}
