package yaml

type ComparableValue interface {
	EquivalentTo(interface{}) bool
}
