package vault

// VaultClient interface for getting secrets
type VaultClient interface {
	GetSecret(ref string) (string, error)
}
