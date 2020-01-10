package server

const (
	BootstrapConfigurationKey               = "bootstrap.enabled"
	CookieAuthenticationKeyConfigurationKey = "cookie.authentication.key"
	CookieEncryptionKeyConfigurationKey     = "cookie.encryption.key"
	OIDCIssuerConfigurationKey              = "oidc.issuer.baseurl"
	Auth0ClientIdConfigurationKey           = "oidc.auth0.clientid"
	Auth0ClientSecretConfigurationKey       = "oidc.auth0.clientsecret"
	SystemBaseUrlConfigurationKey           = "system.baseurl"
)

type ServerConfiguration struct {
	CookieAuthenticationKey []byte // TODO: Encrypt in database
	CookieEncryptionKey     []byte // TODO: Encrypt in database
	OIDCIssuer              string // TODO: Encrypt in database
	Auth0ClientID           string // TODO: Encrypt in database
	Auth0ClientSecret       string // TODO: Encrypt in database
	SystemBaseUrl           string
}
