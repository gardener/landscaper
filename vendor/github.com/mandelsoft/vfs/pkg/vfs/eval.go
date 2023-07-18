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
	"os"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func evalPath(fs FileSystem, path string, exist bool, link ...bool) (string, error) {
	var parsed string
	var dir bool

	vol, elems, rooted := SplitPath(fs, path)
	getlink := true
	if len(link) > 0 {
		getlink = link[0]
	}
outer:
	for {
		parsed = ""
		dir = true

		for i := 0; i < len(elems); i++ {
			e := elems[i]
			next := e
			if len(parsed) > 0 {
				next = parsed + PathSeparatorString + e
			}
			switch e {
			case ".":
				if !dir {
					return "", ErrNotDir
				}
				continue
			case "..":
				if !dir {
					return "", ErrNotDir
				}
				base := Base(nil, parsed)
				if parsed == "" || base == ".." {
					if !rooted {
						parsed = next
					}
				} else {
					parsed = Dir(nil, parsed)
					if parsed == "." {
						parsed = ""
					}
				}
				continue
			}
			p := next
			if rooted {
				p = PathSeparatorString + next
			}
			fi, err := fs.Lstat(p)
			if err != nil {
				if os.IsPermission(err) {
					return "", err
				}
				if exist || !IsErrNotExist(err) {
					return "", NewPathError("", p, err)
				}
				dir = true
				parsed = next
			} else {
				if fi.Mode()&os.ModeType != os.ModeSymlink || (!getlink && i == len(elems)-1) {
					dir = fi.IsDir()
					parsed = next
					continue
				}
				link, err := fs.Readlink(p)
				if err != nil {
					return "", NewPathError("", next, err)
				}
				v, nested, r := SplitPath(fs, link)
				if r {
					elems = append(nested, elems[i+1:]...)
					vol = v
					rooted = r
					continue outer
				}
				elems = append(elems[:i], append(nested, elems[i+1:]...)...)
				i--
			}
		}
		break
	}
	if rooted {
		return vol + PathSeparatorString + parsed, nil
	}
	if len(parsed) == 0 {
		parsed = "."
	}
	return vol + parsed, nil
}
