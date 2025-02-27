package shared

const (
	LabelAppName        = "app.kubernetes.io/name"
	LabelAppInstance    = "app.kubernetes.io/instance"
	LabelVersion        = "app.kubernetes.io/version"
	LabelManagedBy      = "app.kubernetes.io/managed-by"
	LabelValueManagedBy = "landscaper-provider"
	LabelTopology       = "landscaper.gardener.cloud/topology"
	LabelTopologyNs     = "landscaper.gardener.cloud/topology-ns"
)
