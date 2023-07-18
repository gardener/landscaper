/*
 * Copyright 2020 Mandelsoft. All rights reserved.
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

package vfs

import (
	"errors"
	"os"
	"reflect"
)

type ErrorMatcher func(err error) bool

func MatchErr(err error, match ErrorMatcher, base error) bool {
	for err != nil {
		if base == err || (match != nil && match(err)) {
			return true
		}
		switch nerr := err.(type) {
		case interface{ Unwrap() error }:
			err = nerr.Unwrap()
		default:
			err = nil
			v := reflect.ValueOf(nerr)
			if v.Kind() == reflect.Struct {
				f := v.FieldByName("Err")
				if f.IsValid() {
					err, _ = f.Interface().(error)
				}
			}
		}
	}
	return false
}

func IsErrNotDir(err error) bool {
	return MatchErr(err, isUnderlyingErrNotDir, ErrNotDir)
}

func IsErrNotExist(err error) bool {
	if os.IsNotExist(err) {
		return true
	}
	return MatchErr(err, os.IsNotExist, ErrNotExist)
}

func IsErrExist(err error) bool {
	if os.IsExist(err) {
		return true
	}
	return MatchErr(err, os.IsExist, ErrExist)
}

func IsErrReadOnly(err error) bool {
	return MatchErr(err, nil, ErrReadOnly)
}

func NewPathError(op string, path string, err error) error {
	return &os.PathError{Op: op, Path: path, Err: err}
}

var ErrNotDir = errors.New("is no directory")
var ErrNotExist = os.ErrNotExist
var ErrExist = os.ErrExist

var ErrReadOnly = errors.New("filehandle is not writable")
var ErrNotEmpty = errors.New("dir not empty")
