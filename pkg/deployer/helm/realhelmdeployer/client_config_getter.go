package realhelmdeployer

import (
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type ClientConfigGetter struct {
	config    *rest.Config
	namespace string
}

func (c *ClientConfigGetter) ClientConfig() (*rest.Config, error) {
	return c.config, nil
}

func (c *ClientConfigGetter) RawConfig() (clientcmdapi.Config, error) {
	panic("Not implemented")
}

func (c *ClientConfigGetter) Namespace() (string, bool, error) {
	return c.namespace, false, nil
}

func (c *ClientConfigGetter) ConfigAccess() clientcmd.ConfigAccess {
	panic("Not implemented")
}
