package server

const (
	CookieAuthenticationKeyConfigurationKey = "cookie.authentication.key"
	CookieEncryptionKeyConfigurationKey     = "cookie.encryption.key"
)

type ServerConfiguration struct {
	CookieAuthenticationKey []byte // TODO: Encrypt in database
	CookieEncryptionKey     []byte // TODO: Encrypt in database
}
