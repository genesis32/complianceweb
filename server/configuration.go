package server

// The keys in the settings table that corresponse to configuration.
const (
	BootstrapConfigurationKey               = "bootstrap.enabled"
	CookieAuthenticationKeyConfigurationKey = "cookie.authentication.key"
	CookieEncryptionKeyConfigurationKey     = "cookie.encryption.key"
	OIDCIssuerBaseURLConfigurationKey       = "oidc.issuer.baseurl"
	Auth0ClientIDConfigurationKey           = "oidc.auth0.clientid"
	Auth0ClientSecretConfigurationKey       = "oidc.auth0.clientsecret"
	SystemBaseURLConfigurationKey           = "system.baseurl"
)

// ServerConfiguration contains all the database configuration.
type Configuration struct {
	CookieAuthenticationKey []byte // TODO: Encrypt in database
	CookieEncryptionKey     []byte // TODO: Encrypt in database
	OIDCIssuer              string // TODO: Encrypt in database
	Auth0ClientID           string // TODO: Encrypt in database
	Auth0ClientSecret       string // TODO: Encrypt in database
	SystemBaseUrl           string
}
