package filepath

import (
	"errors"
	"os"
	"strings"
)

// SplitVolume splits a path into a volume and a path part.
func SplitVolume(path string) (string, string) {
	vol := VolumeName(path)
	return vol, path[len(vol):]
}

// SplitPath splits a path into a volume, an array of the path segments and a rooted flag.
// The rooted flag is true, if the given path is an absolute one. In this case the
// segment array does not contain a root segment.
func SplitPath(path string) (string, []string, bool) {
	vol, path := SplitVolume(path)
	rest := path
	elems := []string{}
	for rest != "" {
		i := 0
		for i < len(rest) && os.IsPathSeparator(rest[i]) {
			i++
		}
		j := i
		for j < len(rest) && !os.IsPathSeparator(rest[j]) {
			j++
		}
		b := rest[i:j]
		rest = rest[j:]
		if b == "." || b == "" {
			continue
		}
		elems = append(elems, b)
	}
	return vol, elems, strings.HasPrefix(path, PathSeparatorString)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func evalPath(path string, exist bool, link ...bool) (string, error) {
	var parsed string
	var dir bool

	vol, elems, rooted := SplitPath(path)
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
					return "", errors.New("not a directory")
				}
				continue
			case "..":
				if !dir {
					return "", errors.New("not a directory")
				}
				base := Base(parsed)
				if parsed == "" || base == ".." {
					if !rooted {
						parsed = next
					}
				} else {
					parsed = Dir(parsed)
					if parsed == "." {
						parsed = ""
					}
				}
				continue
			}
			p := next
			if rooted {
				p = string(os.PathSeparator) + next
			}
			fi, err := os.Lstat(p)
			if err != nil {
				if os.IsPermission(err) {
					return "", &os.PathError{"", p, err}
				}
				if exist || !os.IsNotExist(err) {
					return "", &os.PathError{"", p, err}
				}
				dir = true
				parsed = next
			} else {
				if fi.Mode()&os.ModeType != os.ModeSymlink || (!getlink && i == len(elems)-1) {
					dir = fi.IsDir()
					parsed = next
					continue
				}
				link, err := os.Readlink(p)
				if err != nil {
					return "", &os.PathError{"", next, err}
				}
				v, nested, r := SplitPath(link)
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
