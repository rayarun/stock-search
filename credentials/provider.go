package credentials

import (
	"fmt"
	"os"
)

// Provider defines the interface for credential providers
type Provider interface {
	GetCredential(key string) (string, error)
}

// EnvProvider retrieves credentials from environment variables
type EnvProvider struct{}

func NewEnvProvider() *EnvProvider {
	return &EnvProvider{}
}

func (p *EnvProvider) GetCredential(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("credential not found: %s", key)
	}
	return value, nil
}

// KMSProvider is a placeholder for KMS-based credential retrieval
// Users can implement this interface to integrate with their KMS solution
// (AWS KMS, Google Cloud KMS, HashiCorp Vault, etc.)
type KMSProvider struct {
	// KMS client configuration would go here
	// Example: kmsClient *kms.Client
	DecryptFunc func(encryptedKey string) (string, error)
}

func NewKMSProvider(decryptFunc func(string) (string, error)) *KMSProvider {
	return &KMSProvider{
		DecryptFunc: decryptFunc,
	}
}

func (p *KMSProvider) GetCredential(key string) (string, error) {
	if p.DecryptFunc == nil {
		return "", fmt.Errorf("KMS decrypt function not configured")
	}

	// In a real implementation, this would:
	// 1. Fetch encrypted credential from KMS
	// 2. Decrypt using KMS service
	// 3. Return decrypted value
	return p.DecryptFunc(key)
}

// StaticProvider for testing with hardcoded credentials
type StaticProvider struct {
	credentials map[string]string
}

func NewStaticProvider(creds map[string]string) *StaticProvider {
	return &StaticProvider{
		credentials: creds,
	}
}

func (p *StaticProvider) GetCredential(key string) (string, error) {
	value, ok := p.credentials[key]
	if !ok {
		return "", fmt.Errorf("credential not found: %s", key)
	}
	return value, nil
}
