package dynaml

import (
	"github.com/mandelsoft/spiff/yaml"
)

var (
	refJobs = ReferenceExpr{[]string{"", "jobs"}}
)

type AutoExpr struct {
	Path []string
}

func (e AutoExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(e.Path) == 3 && e.Path[0] == "resource_pools" && e.Path[2] == "size" {
		jobs, info, found := refJobs.Evaluate(binding, false)
		if !found {
			info.Issue = yaml.NewIssue("no jobs found")
			return nil, info, false
		}

		if !isResolvedValue(jobs) {
			return e, info, true
		}
		jobsList, ok := jobs.([]yaml.Node)
		if !ok {
			return info.Error("jobs must be a list")
		}

		var size int64

		for _, job := range jobsList {
			poolName, ok := yaml.FindString(job, "resource_pool")
			if !ok {
				continue
			}

			if poolName != yaml.PathComponent(e.Path[1]) {
				continue
			}

			instances, ok := yaml.FindInt(job, "instances")
			if !ok {
				return nil, info, false
			}

			size += instances
		}

		return size, info, true
	}

	return info.Error("auto only allowed for size entry in resource pools")
}

func (e AutoExpr) String() string {
	return "auto"
}
