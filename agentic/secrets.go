// Package agentic provides secure secrets management for the agentic orchestration system.
package agentic

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/xerrors"
)

// SecretStore manages encrypted secrets for API keys, credentials, etc.
type SecretStore interface {
	Set(key, value string) error
	Get(key string) (string, error)
	Delete(key string) error
	List() ([]string, error)
}

// EnvSecretStore implements SecretStore using environment variables.
type EnvSecretStore struct{}

func NewEnvSecretStore() *EnvSecretStore {
	return &EnvSecretStore{}
}

func (e *EnvSecretStore) Set(key, value string) error {
	return os.Setenv(key, value)
}

func (e *EnvSecretStore) Get(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", xerrors.New("secret not found")
	}
	return value, nil
}

func (e *EnvSecretStore) Delete(key string) error {
	return os.Unsetenv(key)
}

func (e *EnvSecretStore) List() ([]string, error) {
	env := os.Environ()
	keys := make([]string, 0, len(env))
	for _, pair := range env {
		key := strings.SplitN(pair, "=", 2)[0]
		keys = append(keys, key)
	}
	return keys, nil
}

// FileSecretStore implements SecretStore using encrypted local files.
type FileSecretStore struct {
	filePath string
	password string
}

type encryptedSecrets struct {
	Salt    string            `json:"salt"`
	Nonce   string            `json:"nonce"`
	Data    string            `json:"data"`
	Secrets map[string]string `json:"-"` // Decrypted data
}

func NewFileSecretStore(filePath, password string) *FileSecretStore {
	return &FileSecretStore{
		filePath: filePath,
		password: password,
	}
}

func (f *FileSecretStore) deriveKey(salt []byte) []byte {
	return pbkdf2.Key([]byte(f.password), salt, 4096, 32, sha256.New)
}

func (f *FileSecretStore) loadSecrets() (*encryptedSecrets, error) {
	if _, err := os.Stat(f.filePath); os.IsNotExist(err) {
		return &encryptedSecrets{Secrets: make(map[string]string)}, nil
	}

	data, err := os.ReadFile(f.filePath)
	if err != nil {
		return nil, err
	}

	var secrets encryptedSecrets
	if err := json.Unmarshal(data, &secrets); err != nil {
		return nil, err
	}

	// Decrypt the data
	salt, err := base64.StdEncoding.DecodeString(secrets.Salt)
	if err != nil {
		return nil, err
	}

	nonce, err := base64.StdEncoding.DecodeString(secrets.Nonce)
	if err != nil {
		return nil, err
	}

	ciphertext, err := base64.StdEncoding.DecodeString(secrets.Data)
	if err != nil {
		return nil, err
	}

	key := f.deriveKey(salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	secrets.Secrets = make(map[string]string)
	if err := json.Unmarshal(plaintext, &secrets.Secrets); err != nil {
		return nil, err
	}

	return &secrets, nil
}

func (f *FileSecretStore) saveSecrets(secrets *encryptedSecrets) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(f.filePath), 0700); err != nil {
		return err
	}

	// Encrypt the secrets
	plaintext, err := json.Marshal(secrets.Secrets)
	if err != nil {
		return err
	}

	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return err
	}

	key := f.deriveKey(salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}

	ciphertext := aesGCM.Seal(nil, nonce, plaintext, nil)

	secrets.Salt = base64.StdEncoding.EncodeToString(salt)
	secrets.Nonce = base64.StdEncoding.EncodeToString(nonce)
	secrets.Data = base64.StdEncoding.EncodeToString(ciphertext)

	data, err := json.MarshalIndent(secrets, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(f.filePath, data, 0600)
}

func (f *FileSecretStore) Set(key, value string) error {
	secrets, err := f.loadSecrets()
	if err != nil {
		return err
	}

	secrets.Secrets[key] = value
	return f.saveSecrets(secrets)
}

func (f *FileSecretStore) Get(key string) (string, error) {
	secrets, err := f.loadSecrets()
	if err != nil {
		return "", err
	}

	value, exists := secrets.Secrets[key]
	if !exists {
		return "", xerrors.New("secret not found")
	}

	return value, nil
}

func (f *FileSecretStore) Delete(key string) error {
	secrets, err := f.loadSecrets()
	if err != nil {
		return err
	}

	delete(secrets.Secrets, key)
	return f.saveSecrets(secrets)
}

func (f *FileSecretStore) List() ([]string, error) {
	secrets, err := f.loadSecrets()
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(secrets.Secrets))
	for key := range secrets.Secrets {
		keys = append(keys, key)
	}

	return keys, nil
}

// SecretManager provides a unified interface for multiple secret stores.
type SecretManager struct {
	stores map[string]SecretStore
	logger func(string, ...interface{}) // Optional logger
}

func NewSecretManager() *SecretManager {
	return &SecretManager{
		stores: make(map[string]SecretStore),
	}
}

func (sm *SecretManager) SetLogger(logger func(string, ...interface{})) {
	sm.logger = logger
}

func (sm *SecretManager) AddStore(name string, store SecretStore) {
	sm.stores[name] = store
}

func (sm *SecretManager) Get(key string) (string, error) {
	// Try environment variables first
	if envStore, exists := sm.stores["env"]; exists {
		if value, err := envStore.Get(key); err == nil {
			return value, nil
		}
	}

	// Try other stores
	for name, store := range sm.stores {
		if name == "env" {
			continue // Already tried
		}
		if value, err := store.Get(key); err == nil {
			return value, nil
		}
	}

	return "", xerrors.Errorf("secret '%s' not found in any store", key)
}

func (sm *SecretManager) Set(storeName, key, value string) error {
	store, exists := sm.stores[storeName]
	if !exists {
		return xerrors.Errorf("store '%s' not found", storeName)
	}

	// Never log secret values
	if sm.logger != nil {
		sm.logger("Setting secret key '%s' in store '%s'", key, storeName)
	}

	return store.Set(key, value)
}

func (sm *SecretManager) Delete(storeName, key string) error {
	store, exists := sm.stores[storeName]
	if !exists {
		return xerrors.Errorf("store '%s' not found", storeName)
	}

	if sm.logger != nil {
		sm.logger("Deleting secret key '%s' from store '%s'", key, storeName)
	}

	return store.Delete(key)
}

// GetCredentials retrieves common credential sets safely.
func (sm *SecretManager) GetCredentials(service string) (map[string]string, error) {
	creds := make(map[string]string)

	switch service {
	case "github":
		if clientID, err := sm.Get("GITHUB_CLIENT_ID"); err == nil {
			creds["client_id"] = clientID
		}
		if clientSecret, err := sm.Get("GITHUB_CLIENT_SECRET"); err == nil {
			creds["client_secret"] = clientSecret
		}
		if token, err := sm.Get("GITHUB_TOKEN"); err == nil {
			creds["token"] = token
		}

	case "proxmox":
		if url, err := sm.Get("PROXMOX_URL"); err == nil {
			creds["url"] = url
		}
		if username, err := sm.Get("PROXMOX_USERNAME"); err == nil {
			creds["username"] = username
		}
		if password, err := sm.Get("PROXMOX_PASSWORD"); err == nil {
			creds["password"] = password
		}
		if token, err := sm.Get("PROXMOX_TOKEN"); err == nil {
			creds["token"] = token
		}

	case "cloudflare":
		if apiKey, err := sm.Get("CLOUDFLARE_API_KEY"); err == nil {
			creds["api_key"] = apiKey
		}
		if email, err := sm.Get("CLOUDFLARE_EMAIL"); err == nil {
			creds["email"] = email
		}
		if token, err := sm.Get("CLOUDFLARE_TOKEN"); err == nil {
			creds["token"] = token
		}

	case "nix":
		if nixPath, err := sm.Get("NIX_PATH"); err == nil {
			creds["nix_path"] = nixPath
		}
		if signingKey, err := sm.Get("NIX_SIGNING_KEY"); err == nil {
			creds["signing_key"] = signingKey
		}
		if substitutes, err := sm.Get("NIX_SUBSTITUTES"); err == nil {
			creds["substitutes"] = substitutes
		}
		if trustedKeys, err := sm.Get("NIX_TRUSTED_KEYS"); err == nil {
			creds["trusted_keys"] = trustedKeys
		}
		if remoteHosts, err := sm.Get("NIX_REMOTE_HOSTS"); err == nil {
			creds["remote_hosts"] = remoteHosts
		}

	default:
		return nil, xerrors.Errorf("unknown service: %s", service)
	}

	if len(creds) == 0 {
		return nil, xerrors.Errorf("no credentials found for service: %s", service)
	}

	return creds, nil
}
