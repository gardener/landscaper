package cd_facade

// Auth describes a interface of the dockerconfigtypes.Auth struct
type Auth interface {
	GetUsername() string
	GetPassword() string
	GetAuth() string

	// GetIdentityToken is used to authenticate the user and get
	// an access token for the registry.
	GetIdentityToken() string

	// GetRegistryToken is a bearer token to be sent to a registry
	GetRegistryToken() string
}
