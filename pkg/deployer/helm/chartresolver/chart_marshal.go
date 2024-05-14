package chartresolver

import (
	"encoding/json"

	"helm.sh/helm/v3/pkg/chart"
)

func MarshalChart(chart *chart.Chart) ([]byte, error) {
	tree := treeFromChart(chart)
	return json.Marshal(tree)
}

func UnmarshalChart(bytes []byte) (*chart.Chart, error) {
	tree := &ChartTree{}
	if err := json.Unmarshal(bytes, tree); err != nil {
		return nil, err
	}
	return chartFromTree(tree), nil
}

// ChartTree is a tree containing a chart and its subcharts (recursively),
// both in public fields so that they are respected during marshaling.
type ChartTree struct {
	Chart    *chart.Chart `json:"chart,omitempty"`
	SubTrees []*ChartTree `json:"subTrees,omitempty"`
}

func treeFromChart(chart *chart.Chart) *ChartTree {
	tree := &ChartTree{
		Chart: chart,
	}

	subCharts := chart.Dependencies()
	if len(subCharts) > 0 {
		tree.SubTrees = make([]*ChartTree, len(subCharts))
		for i := range subCharts {
			tree.SubTrees[i] = treeFromChart(subCharts[i])
		}
	}

	return tree
}

func chartFromTree(tree *ChartTree) *chart.Chart {
	ch := tree.Chart

	if len(tree.SubTrees) > 0 {
		subCharts := make([]*chart.Chart, len(tree.SubTrees))
		for i := range tree.SubTrees {
			subCharts[i] = chartFromTree(tree.SubTrees[i])
		}
		ch.AddDependency(subCharts...)
	}

	return ch
}
