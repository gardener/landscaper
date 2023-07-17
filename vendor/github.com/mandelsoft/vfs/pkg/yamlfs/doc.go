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

// Package yamlfs provides a virtual filesystem based on the structure and
// content of a yaml document.
//
// Hereby maps are used to represent the directory hierarchy. Content of
// a field will be offered as file content.
// To represent more complex cases a map inclusing the field `$type` is
// not interpreted as directory by default, but according to the
// value of the type attribute.
// - type `directory`: the field `value` is used a directory content
// - type `file`: the field `value` is used as file content
// - type `symlink`: the field `value` is used symlink content
// - type `yaml`: the field  `value` is used to provide file content
//   that is provided as yaml file data (read and write)
// - type `json`: the field  `value` is used to provide file content
//   that is provided as json file data (read and write)
//
// string data starting with a line `---Start Binary---`and finished with a
// line ` ---End Binary---` interpretes the data in-between as bas64 encoded
// binary data. The latest version is able to handle binary data directly
// as described by the yaml format for reading and writing.
package yamlfs
