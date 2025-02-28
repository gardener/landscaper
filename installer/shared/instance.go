package shared

import "fmt"

// Instance identifies a landscaper installation for an update or delete operation, e.g. "test0001-abcdefgh".
type Instance string

// Namespace is the namespace on the landscaper host resp. resource cluster where objects will be installed.
func (i Instance) Namespace() string {
	return fmt.Sprintf("ls-system-%s", i)
}
