package adapters

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/snowflakedb/gosnowflake"
	"github.com/youmark/pkcs8"
)

type snowflakeDSNOptions struct {
	privateKeyPath          string
	privateKeyEnv           string
	privateKeyPassphrase    string
	privateKeyPassphraseEnv string
}

func prepareSnowflakeConfig(rawURL string) (*gosnowflake.Config, error) {
	dsn, opts, err := extractSnowflakeDSNOptions(rawURL)
	if err != nil {
		return nil, err
	}

	if opts.hasPrivateKey() {
		dsn, err = setSnowflakeQueryValue(dsn, "authenticator", gosnowflake.AuthTypeJwt.String())
		if err != nil {
			return nil, err
		}
	}

	config, err := gosnowflake.ParseDSN(dsn)
	if err != nil {
		return nil, err
	}

	if opts.hasPrivateKey() {
		privateKey, err := loadSnowflakePrivateKey(opts)
		if err != nil {
			return nil, err
		}
		config.PrivateKey = privateKey
		config.Authenticator = gosnowflake.AuthTypeJwt
	}

	return config, nil
}

func extractSnowflakeDSNOptions(rawURL string) (string, *snowflakeDSNOptions, error) {
	opts := &snowflakeDSNOptions{}

	base, rawQuery, hasQuery := strings.Cut(rawURL, "?")
	if !hasQuery {
		return rawURL, opts, nil
	}

	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return "", nil, err
	}

	opts.privateKeyPath = popQueryValue(values, "privateKeyPath", "private_key_path")
	opts.privateKeyEnv = popQueryValue(values, "privateKeyEnv", "private_key_env")
	opts.privateKeyPassphrase = popQueryValue(values, "privateKeyPassphrase", "privateKeyPassword", "private_key_passphrase", "private_key_password")
	opts.privateKeyPassphraseEnv = popQueryValue(values, "privateKeyPassphraseEnv", "privateKeyPasswordEnv", "private_key_passphrase_env", "private_key_password_env")

	query := values.Encode()
	if query == "" {
		return base, opts, nil
	}
	return base + "?" + query, opts, nil
}

func (o *snowflakeDSNOptions) hasPrivateKey() bool {
	return o.privateKeyPath != "" || o.privateKeyEnv != ""
}

func setSnowflakeQueryValue(dsn, key, value string) (string, error) {
	base, rawQuery, _ := strings.Cut(dsn, "?")
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return "", err
	}
	values.Set(key, value)
	return base + "?" + values.Encode(), nil
}

func popQueryValue(values url.Values, names ...string) string {
	for _, name := range names {
		if got, ok := values[name]; ok {
			delete(values, name)
			if len(got) > 0 {
				return got[0]
			}
			return ""
		}
	}
	return ""
}

func loadSnowflakePrivateKey(opts *snowflakeDSNOptions) (*rsa.PrivateKey, error) {
	if opts.privateKeyPath != "" && opts.privateKeyEnv != "" {
		return nil, errors.New("only one of privateKeyPath or privateKeyEnv can be set")
	}
	if opts.privateKeyPassphrase != "" && opts.privateKeyPassphraseEnv != "" {
		return nil, errors.New("only one of privateKeyPassphrase or privateKeyPassphraseEnv can be set")
	}

	var bytes []byte
	var err error
	if opts.privateKeyPath != "" {
		bytes, err = os.ReadFile(expandUserPath(opts.privateKeyPath))
	} else {
		bytes, err = privateKeyBytesFromEnv(opts.privateKeyEnv)
	}
	if err != nil {
		return nil, err
	}

	passphrase := opts.privateKeyPassphrase
	if passphrase == "" && opts.privateKeyPassphraseEnv != "" {
		passphrase = os.Getenv(opts.privateKeyPassphraseEnv)
	}

	return parseSnowflakePrivateKeyPEM(bytes, passphrase)
}

func privateKeyBytesFromEnv(name string) ([]byte, error) {
	value := os.Getenv(name)
	if value == "" {
		return nil, fmt.Errorf("private key env var %q is empty", name)
	}
	if strings.Contains(value, "-----BEGIN") {
		return []byte(value), nil
	}
	return os.ReadFile(expandUserPath(value))
}

func expandUserPath(path string) string {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return home + path[1:]
		}
	}
	return path
}

func parseSnowflakePrivateKeyPEM(bytes []byte, passphrase string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(bytes)
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the private key")
	}

	if passphrase != "" {
		key, err := pkcs8.ParsePKCS8PrivateKeyRSA(block.Bytes, []byte(passphrase))
		if err == nil {
			return key, nil
		}

		if x509.IsEncryptedPEMBlock(block) {
			decrypted, decryptErr := x509.DecryptPEMBlock(block, []byte(passphrase))
			if decryptErr != nil {
				return nil, decryptErr
			}
			return parseSnowflakeRSAPrivateKey(decrypted)
		}

		return nil, err
	}

	if key, err := pkcs8.ParsePKCS8PrivateKeyRSA(block.Bytes); err == nil {
		return key, nil
	}
	return parseSnowflakeRSAPrivateKey(block.Bytes)
}

func parseSnowflakeRSAPrivateKey(der []byte) (*rsa.PrivateKey, error) {
	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}

	key, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, err
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("expected RSA private key, got %T", key)
	}
	return rsaKey, nil
}
