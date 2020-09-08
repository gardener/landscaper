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

package test

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"

	. "github.com/onsi/gomega"

	"github.com/mandelsoft/vfs/pkg/vfs"
)

func List(fs vfs.FileSystem, path string) ([]string, error) {
	f, err := fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.Readdirnames(0)
}

func ExpectFolders(fs vfs.FileSystem, path string, names []string, err error) {
	found, ferr := List(fs, path)
	if err == nil {
		Expect(ferr).To(BeNil())
	} else {
		Expect(ferr).To(Equal(err))
		return
	}
	if names == nil {
		names = []string{}
	}
	Expect(found).To(Equal(names))
}

func ExpectRead(f io.Reader, content interface{}) {
	buf := bytes.Buffer{}
	rbuf := [10]byte{}

	for {
		n, err := f.Read(rbuf[:])
		if n > 0 {
			buf.Write(rbuf[:n])
		}
		if err == io.EOF {
			break
		}
		Expect(err).To(Succeed())
	}
	if s, ok := content.(string); ok {
		Expect(string(buf.Bytes())).To(Equal(s))
	} else {
		Expect(buf.Bytes()).To(Equal(content))
	}
}

func ExpectFileCreate(fs vfs.FileSystem, path string, content []byte, err error) {
	f, ferr := fs.Create(path)
	if err == nil {
		Expect(ferr).To(BeNil())
	} else {
		Expect(ferr).To(Equal(err))
		return
	}
	if content != nil {
		Expect(f.Write(content)).To(Equal(len(content)))
	}
	Expect(f.Close()).To(Succeed())

	d, b := vfs.Split(fs, path)
	if d == "" {
		d = "."
	}
	Expect(List(fs, d)).Should(ContainElement(b))

	if content != nil {
		f, err := fs.Open(path)
		Expect(err).To(BeNil())

		Expect(ioutil.ReadAll(f)).To(Equal(content))
		Expect(f.Close()).To(Succeed())
	}
}

func ExpectFileWrite(fs vfs.FileSystem, path string, flags int, content interface{}, check ...bool) {
	f, err := fs.OpenFile(path, flags|os.O_WRONLY, os.ModePerm)
	Expect(err).To(Succeed())

	var b []byte
	if s, ok := content.(string); ok {
		b = []byte(s)
	} else {
		b = content.([]byte)
	}
	n, err := f.Write(b)
	Expect(err).To(Succeed())
	Expect(n).To(Equal(n))
	Expect(f.Close()).To(Succeed())
	if len(check) == 0 || check[0] {
		ExpectFileContent(fs, path, content)
	}
}

func ExpectFileContent(fs vfs.FileSystem, path string, content interface{}) {
	f, err := fs.Open(path)
	Expect(err).To(Succeed())
	ExpectRead(f, content)
	Expect(f.Close()).To(Succeed())
}

type ErrMatcher func(err error) bool

func ExpectMatchedErr(m ErrMatcher, args ...interface{}) {
	Expect(m(args[len(args)-1].(error))).To(BeTrue())
}

func ExpectErr(args ...interface{}) {
	Expect(args[len(args)-1]).To(HaveOccurred())
}

func ExpectSucceeded(args ...interface{}) {
	Expect(args[len(args)-1]).To(Succeed())
}
