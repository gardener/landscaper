package dynaml

import (
	"net"
	"strings"

	"github.com/mandelsoft/spiff/yaml"
)

var (
	refName      = ReferenceExpr{[]string{"name"}}
	refInstances = ReferenceExpr{[]string{"instances"}}
)

func func_static_ips(arguments []Expression, binding Binding) (interface{}, EvaluationInfo, bool) {

	indices := []int{}
	for _, arg := range arguments {
		index, info, ok := arg.Evaluate(binding, false)
		if !ok {
			return nil, info, false
		}

		index64, ok := index.(int64)
		if ok {
			indices = append(indices, int(index64))
		} else {
			list, ok := index.([]yaml.Node)
			if !ok {
				return info.Error("arguments to static_ips must be integer or list of integers")
			}
			_, info, ok = getIndices(&indices, list, info)
		}
	}

	return generateStaticIPs(binding, indices)
}

func getIndices(indices *[]int, list []yaml.Node, info EvaluationInfo) (interface{}, EvaluationInfo, bool) {
	for _, elem := range list {
		if elem != nil && elem.Value() != nil {
			index64, ok := elem.Value().(int64)
			if ok {
				if index64 < 0 {
					return info.Error("negative ip indices are not allowed: %d", index64)
				}
				*indices = append(*indices, int(index64))
			} else {
				list, ok := elem.Value().([]yaml.Node)
				if !ok {
					return info.Error("arguments to static_ips must be integer or list of integers")
				}
				_, info, ok = getIndices(indices, list, info)
				if !ok {
					return nil, info, false
				}
			}
		}
	}
	return nil, info, true
}

func generateStaticIPs(binding Binding, indices []int) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(indices) == 0 {
		return nil, info, false
	}

	ranges, info, ok := findStaticIPRanges(binding)
	if !ok || ranges == nil {
		return nil, info, ok
	}

	instanceCountP, info, ok := findInstanceCount(binding)
	if !ok || instanceCountP == nil {
		return nil, info, ok
	}
	instanceCount := int(*instanceCountP)
	ipPool, ok := staticIPPool(ranges)
	if !ok {
		return nil, info, false
	}

	ips := []yaml.Node{}
	for _, i := range indices {
		if len(ipPool) <= i {
			return nil, info, false
		}

		ips = append(ips, NewNode(ipPool[i].String(), binding))
	}

	if len(ips) < instanceCount {
		return info.Error("too less static IPs for %d instances", instanceCount)
	}

	return ips[:instanceCount], info, true
}

func findInstanceCount(binding Binding) (*int64, EvaluationInfo, bool) {
	nearestInstances, info, found := refInstances.Evaluate(binding, false)
	if !found || isExpression(nearestInstances) {
		return nil, info, false
	}

	instances, ok := nearestInstances.(int64)
	return &instances, info, ok
}

func findStaticIPRanges(binding Binding) ([]string, EvaluationInfo, bool) {
	nearestNetworkName, info, found := refName.Evaluate(binding, false)
	if !found || isExpression(nearestNetworkName) {
		return nil, info, found
	}

	networkName, ok := nearestNetworkName.(string)
	if !ok {
		info.Error("name field must be string")
		return nil, info, false
	}

	subnetsRef := ReferenceExpr{[]string{"", "networks", networkName, "subnets"}}
	subnets, info, found := subnetsRef.Evaluate(binding, false)

	if !found {
		return nil, info, false
	}
	if isExpression(subnets) {
		return nil, info, true
	}

	subnetsList, ok := subnets.([]yaml.Node)
	if !ok {
		info.Error("subnets field must be a list")
		return nil, info, false
	}

	allRanges := []string{}

	for _, subnet := range subnetsList {
		subnetMap, ok := subnet.Value().(map[string]yaml.Node)
		if !ok {
			info.Error("subnet must be a map")
			return nil, info, false
		}

		static, ok := subnetMap["static"]

		if !ok {
			info.Error("no static ips for network %s", networkName)
			return nil, info, false
		}

		staticList, ok := static.Value().([]yaml.Node)
		if !ok {
			info.Issue = yaml.NewIssue("static ips for network %s must be a list", networkName)
			return nil, info, false
		}

		ranges := make([]string, len(staticList))

		for i, r := range staticList {
			ipsString, ok := r.Value().(string)
			if !ok {
				info.Error("invalid entry for static ips for network %s", networkName)
				return nil, info, false
			}

			ranges[i] = ipsString
		}

		allRanges = append(allRanges, ranges...)
	}

	return allRanges, info, true
}

func staticIPPool(ranges []string) ([]net.IP, bool) {
	ipPool := []net.IP{}

	for _, r := range ranges {
		segments := strings.Split(r, "-")
		if len(segments) == 0 {
			return nil, false
		}

		var start, end net.IP

		start = net.ParseIP(strings.Trim(segments[0], " "))

		if len(segments) == 1 {
			end = start
		} else {
			end = net.ParseIP(strings.Trim(segments[1], " "))
		}

		ipPool = append(ipPool, ipRange(start, end)...)
	}

	return ipPool, true
}

func ipRange(a, b net.IP) []net.IP {
	prev := a

	ips := []net.IP{a}

	for !prev.Equal(b) {
		next := net.ParseIP(prev.String())
		inc(next)
		ips = append(ips, next)
		prev = next
	}

	return ips
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
