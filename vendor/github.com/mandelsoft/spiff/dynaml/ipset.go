package dynaml

import (
	"bytes"
	"net"
	"strings"

	"github.com/mandelsoft/spiff/debug"
	"github.com/mandelsoft/spiff/yaml"
)

type IPRange interface {
	GetSize() int64
	GetIP(int64) net.IP
}

type iprange struct {
	start net.IP
	end   net.IP
	size  int64
}

type cidrrange struct {
	net.IPNet
}

func map_ip_ranges(ranges []string) ([]IPRange, EvaluationInfo, bool) {
	ipPool := []IPRange{}

	info := DefaultInfo()
	for _, r := range ranges {
		segments := strings.Split(r, "-")
		debug.Debug("ipset: found range segments '%s': %d %+v", r, len(segments), segments)
		if len(segments) == 0 {
			info.SetError("empty range")
			return nil, info, false
		}

		var start, end net.IP
		var ipr IPRange

		if len(segments) == 1 {
			_, cidr, err := net.ParseCIDR(r)

			if err == nil {
				ipr = &cidrrange{*cidr}
			} else {
				start = net.ParseIP(strings.Trim(segments[0], " "))
				if start == nil {
					info.SetError("invalid IP '%s'", segments[0])
					return nil, info, false
				}
				ipr = &iprange{start, start, int64(1)}
			}
		} else {
			start = net.ParseIP(strings.Trim(segments[0], " "))
			if start == nil {
				info.SetError("invalid IP '%s'", segments[0])
				return nil, info, false
			}
			end = net.ParseIP(strings.Trim(segments[1], " "))
			if end == nil {
				info.SetError("invalid IP '%s'", segments[1])
				return nil, info, false
			}
			if len(start) != len(end) {
				info.SetError("IP type mismatch")
				return nil, info, false
			}
			if bytes.Compare(start, end) > 0 {
				info.SetError("invalid IP range: start (%s) larger than end (%s)", segments[0], segments[1])
				return nil, info, false
			}
			ipr = &iprange{start, end, int64(0)}
		}

		ipPool = append(ipPool, ipr)
	}

	return ipPool, info, true
}

func (i *iprange) GetSize() int64 {
	if i.size == 0 {
		i.size = DiffIP(i.end, i.start) + 1
	}
	debug.Debug("sizeof(%s-%s)=%d", i.start, i.end, i.size)
	return i.size
}

func (i *iprange) GetIP(index int64) net.IP {
	if index < 0 || index >= i.GetSize() {
		return nil
	}
	ip := make(net.IP, len(i.start))
	copy(ip, i.start)
	return IPAdd(ip, int64(index))
}

func (i *cidrrange) GetSize() int64 {
	ones, _ := i.Mask.Size()
	return int64(1 << (32 - uint32(ones)))
}

func (i *cidrrange) GetIP(index int64) net.IP {
	if index < 0 || index >= i.GetSize() {
		return nil
	}
	return IPAdd(i.IP.Mask(i.Mask), int64(index))
}

func func_ipset(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) < 2 {
		return info.Error("at least 2 argument expected (ipranges, size, optional: ip indices")
	}

	var ranges []IPRange

	s, ok := arguments[0].(string)
	if ok {
		ranges, info, ok = map_ip_ranges([]string{s})
		if !ok {
			return nil, info, false
		}
	} else {
		list, ok := arguments[0].([]yaml.Node)
		if !ok {
			return info.Error("ip range or range list expected as first argument")
		}
		rlist := make([]string, len(list))
		for i, v := range list {
			rlist[i], ok = value(v).(string)
			if !ok {
				return info.Error("string entry at ip range list index %d", i)
			}
		}
		ranges, info, ok = map_ip_ranges(rlist)
		if !ok {
			return nil, info, false
		}
	}

	indices := []int{}
	if len(arguments) > 2 {
		for _, index := range arguments[2:] {
			index64, ok := index.(int64)
			if ok {
				if index64 < 0 {
					return info.Error("negative ip indices are not allowed: %d", index64)
				}
				indices = append(indices, int(index64))
			} else {
				list, ok := index.([]yaml.Node)
				if !ok {
					return info.Error("arguments to static_ips must be integer or list of integers")
				}
				_, info, ok = getIndices(&indices, list, info)
				if !ok {
					return nil, info, false
				}
			}
		}
	}

	num, ok := arguments[1].(int64)
	if !ok {
		return info.Error("number of IPs in set expected as second argument")
	}

	if len(arguments) > 2 && int64(len(indices)) < num {
		return info.Error("too many required entries (%d > %d available indices)",
			num, len(indices))
	}

	debug.Debug("ipset: request %d IP(s)", num)
	result := make([]yaml.Node, num)

	for i := 0; i < int(num); i++ {
		index := i
		if len(arguments) > 2 {
			index = indices[i]
		}

		var offset int64
		offset = 0
		for j, r := range ranges {
			if int64(index) < offset+r.GetSize() {
				ip := r.GetIP(int64(index) - offset).String()
				debug.Debug("ipset: get %d from range %d: %s",
					int64(index)-offset, j, ip)
				result[i] = NewNode(ip, nil)
				break
			}
			debug.Debug("ipset: skipping range %d: offset %d size %d",
				j, offset, r.GetSize())
			offset += r.GetSize()
		}
		if result[i] == nil {
			return info.Error("ip index %d (%d) out of range (%d IP(s) available in ranges)",
				i, index, offset)
		}
	}
	return result, info, true
}
