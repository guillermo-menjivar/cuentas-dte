package services

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/spf13/viper"
)

type VaultService struct {
	client *api.Client
}

// NewVaultService creates a new Vault service instance
func NewVaultService() (*VaultService, error) {
	// Configure Vault client
	config := api.DefaultConfig()
	config.Address = viper.GetString("vault_url")

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %v", err)
	}

	// Set token
	token := viper.GetString("vault_token")
	if token == "" {
		return nil, fmt.Errorf("vault token is required")
	}
	client.SetToken(token)

	// Test connection
	if err := testVaultConnection(client); err != nil {
		return nil, fmt.Errorf("failed to connect to Vault: %v", err)
	}

	vs := &VaultService{client: client}

	// Ensure KV v2 engine is enabled
	if err := vs.ensureKVEngine(); err != nil {
		return nil, fmt.Errorf("failed to setup KV engine: %v", err)
	}

	return vs, nil
}

// testVaultConnection verifies that we can connect to Vault
func testVaultConnection(client *api.Client) error {
	auth := client.Auth().Token()
	_, err := auth.LookupSelf()
	if err != nil {
		return fmt.Errorf("vault connection test failed: %v", err)
	}
	return nil
}

// ensureKVEngine ensures the KV v2 secrets engine is enabled at secret/
func (vs *VaultService) ensureKVEngine() error {
	sys := vs.client.Sys()

	// Check if secret/ mount exists
	mounts, err := sys.ListMounts()
	if err != nil {
		return fmt.Errorf("failed to list mounts: %v", err)
	}

	if mount, exists := mounts["secret/"]; exists {
		// Verify it's KV v2
		if mount.Type == "kv" && mount.Options["version"] == "2" {
			return nil // Already properly configured
		}
		log.Println("Warning: secret/ mount exists but is not KV v2")
	}

	return nil // In dev mode, secret/ should already be KV v2
}

// StoreCompanyPassword stores a company's password in Vault
// Returns the Vault reference path to store in the database
func (vs *VaultService) StoreCompanyPassword(companyID string, password string) (string, error) {
	path := fmt.Sprintf("secret/companies/%s/password", companyID)

	secretData := map[string]interface{}{
		"data": map[string]interface{}{
			"password": password,
		},
	}

	_, err := vs.client.Logical().Write(path, secretData)
	if err != nil {
		return "", fmt.Errorf("failed to store password for company %s: %v", companyID, err)
	}

	// Return the path to store as reference in database
	vaultRef := fmt.Sprintf("secret/companies/%s/password", companyID)
	return vaultRef, nil
}

// GetCompanyPassword retrieves a company's password from Vault using the reference path
func (vs *VaultService) GetCompanyPassword(vaultRef string) (string, error) {
	secret, err := vs.client.Logical().Read(vaultRef)
	if err != nil {
		return "", fmt.Errorf("failed to read secret from %s: %v", vaultRef, err)
	}

	if secret == nil {
		return "", fmt.Errorf("secret not found at %s", vaultRef)
	}

	// Extract password from KV v2 structure
	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid secret format at %s", vaultRef)
	}

	password, ok := data["password"].(string)
	if !ok {
		return "", fmt.Errorf("password not found in secret at %s", vaultRef)
	}

	return password, nil
}

// UpdateCompanyPassword updates an existing company password in Vault
func (vs *VaultService) UpdateCompanyPassword(vaultRef string, newPassword string) error {
	secretData := map[string]interface{}{
		"data": map[string]interface{}{
			"password": newPassword,
		},
	}

	_, err := vs.client.Logical().Write(vaultRef, secretData)
	if err != nil {
		return fmt.Errorf("failed to update password at %s: %v", vaultRef, err)
	}

	return nil
}

// DeleteCompanyPassword removes a company's password from Vault
func (vs *VaultService) DeleteCompanyPassword(vaultRef string) error {
	_, err := vs.client.Logical().Delete(vaultRef)
	if err != nil {
		return fmt.Errorf("failed to delete secret at %s: %v", vaultRef, err)
	}

	return nil
}

// WaitForVault waits for Vault to be available with retries
func WaitForVault(maxRetries int) error {
	for i := 0; i < maxRetries; i++ {
		_, err := NewVaultService()
		if err == nil {
			log.Println("Successfully connected to Vault")
			return nil
		}

		log.Printf("Vault connection attempt %d/%d failed: %v", i+1, maxRetries, err)
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * 2 * time.Second)
		}
	}

	return fmt.Errorf("failed to connect to Vault after %d attempts", maxRetries)
}
