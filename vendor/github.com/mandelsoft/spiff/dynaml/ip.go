package dynaml

import (
	"net"

	"github.com/mandelsoft/spiff/yaml"
)

func func_ip(op func(ip net.IP, cidr *net.IPNet) interface{}, arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) != 1 {
		info.Issue = yaml.NewIssue("only one argument expected for CIDR function")
		return nil, info, false
	}

	str, ok := arguments[0].(string)
	if !ok {
		info.Issue = yaml.NewIssue("CIDR argument required")
		return nil, info, false
	}

	ip, cidr, err := net.ParseCIDR(str)

	if err != nil {
		info.Issue = yaml.NewIssue("CIDR argument required")
		return nil, info, false
	}

	return op(ip, cidr), info, true
}

func func_containsIP(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) != 2 {
		info.Issue = yaml.NewIssue("contains_ip requires CIDR and IP argument")
		return nil, info, false
	}

	str, ok := arguments[0].(string)
	if !ok {
		info.Issue = yaml.NewIssue("CIDR required as first argument")
		return nil, info, false
	}

	_, cidr, err := net.ParseCIDR(str)

	if err != nil {
		info.Issue = yaml.NewIssue("CIDR argument required: %s", err)
		return nil, info, false
	}

	str, ok = arguments[1].(string)
	if !ok {
		info.Issue = yaml.NewIssue("IP required as second argument")
		return nil, info, false
	}

	ip := net.ParseIP(str)
	if ip == nil {
		info.Issue = yaml.NewIssue("IP argument required: %s", str)
		return nil, info, false
	}
	return cidr.Contains(ip), info, true
}

func func_minIP(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	return func_ip(func(ip net.IP, cidr *net.IPNet) interface{} {
		return ip.Mask(cidr.Mask).String()
	}, arguments, binding)
}

func func_maxIP(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	return func_ip(func(ip net.IP, cidr *net.IPNet) interface{} {
		return MaxIP(cidr).String()
	}, arguments, binding)
}

func func_numIP(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	return func_ip(func(ip net.IP, cidr *net.IPNet) interface{} {
		ones, _ := cidr.Mask.Size()
		return int64(1 << (32 - uint32(ones)))
	}, arguments, binding)
}

func SubIP(ip net.IP, mask net.IPMask) net.IP {
	m := ip.Mask(mask)
	out := make(net.IP, len(ip))
	for i, v := range ip {
		j := len(ip) - i
		if j > len(m) {
			out[i] = v
		} else {
			out[i] = v &^ m[len(m)-j]
		}
	}
	return out
}

func MaxIP(cidr *net.IPNet) net.IP {
	ip := cidr.IP.Mask(cidr.Mask)
	mask := cidr.Mask
	out := make(net.IP, len(ip))
	for i, v := range ip {
		j := len(ip) - i
		if j > len(mask) {
			out[i] = v
		} else {
			out[i] = v | ^mask[len(mask)-j]
		}
	}
	return out
}

func DiffIP(a, b net.IP) int64 {
	var d int64

	for i, _ := range a {
		db := int64(a[i]) - int64(b[i])
		d = d*256 + db
	}
	return d
}
