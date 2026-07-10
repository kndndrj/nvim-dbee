package adapters

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/snowflakedb/gosnowflake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/youmark/pkcs8"
)

func TestExtractSnowflakeDSNOptions(t *testing.T) {
	got, opts, err := extractSnowflakeDSNOptions("user@account/db/schema?warehouse=wh&privateKeyPath=/tmp/key.p8&privateKeyPassphraseEnv=KEY_PASSWORD")
	require.NoError(t, err)

	assert.Equal(t, "user@account/db/schema?warehouse=wh", got)
	assert.Equal(t, "/tmp/key.p8", opts.privateKeyPath)
	assert.Equal(t, "KEY_PASSWORD", opts.privateKeyPassphraseEnv)
}

func TestPrepareSnowflakeConfigWithPrivateKeyPath(t *testing.T) {
	privateKeyPath := writeTestPrivateKey(t, makeTestPrivateKeyPEM(t))

	cfg, err := prepareSnowflakeConfig("user@account/db/schema?privateKeyPath=" + privateKeyPath)
	require.NoError(t, err)

	assert.Equal(t, gosnowflake.AuthTypeJwt, cfg.Authenticator)
	assert.NotNil(t, cfg.PrivateKey)
}

func TestPrepareSnowflakeConfigWithPrivateKeyPathExpandsHome(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	privateKeyPath := filepath.Join(home, ".dbee-test-snowflake-key.p8")
	require.NoError(t, os.WriteFile(privateKeyPath, makeTestPrivateKeyPEM(t), 0o600))
	t.Cleanup(func() {
		_ = os.Remove(privateKeyPath)
	})

	cfg, err := prepareSnowflakeConfig("user@account/db/schema?privateKeyPath=~/.dbee-test-snowflake-key.p8")
	require.NoError(t, err)

	assert.Equal(t, gosnowflake.AuthTypeJwt, cfg.Authenticator)
	assert.NotNil(t, cfg.PrivateKey)
}

func TestPrepareSnowflakeConfigWithPrivateKeyEnvContents(t *testing.T) {
	privateKeyPEM := makeTestPrivateKeyPEM(t)
	t.Setenv("DBEE_TEST_SNOWFLAKE_PRIVATE_KEY", string(privateKeyPEM))

	cfg, err := prepareSnowflakeConfig("user@account/db/schema?privateKeyEnv=DBEE_TEST_SNOWFLAKE_PRIVATE_KEY")
	require.NoError(t, err)

	assert.Equal(t, gosnowflake.AuthTypeJwt, cfg.Authenticator)
	assert.NotNil(t, cfg.PrivateKey)
}

func TestPrepareSnowflakeConfigWithPrivateKeyEnvPathExpandsHome(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	privateKeyPath := filepath.Join(home, ".dbee-test-snowflake-key-env.p8")
	require.NoError(t, os.WriteFile(privateKeyPath, makeTestPrivateKeyPEM(t), 0o600))
	t.Cleanup(func() {
		_ = os.Remove(privateKeyPath)
	})
	t.Setenv("DBEE_TEST_SNOWFLAKE_PRIVATE_KEY_PATH", "~/.dbee-test-snowflake-key-env.p8")

	cfg, err := prepareSnowflakeConfig("user@account/db/schema?privateKeyEnv=DBEE_TEST_SNOWFLAKE_PRIVATE_KEY_PATH")
	require.NoError(t, err)

	assert.Equal(t, gosnowflake.AuthTypeJwt, cfg.Authenticator)
	assert.NotNil(t, cfg.PrivateKey)
}

func TestPrepareSnowflakeConfigWithEncryptedPrivateKeyPath(t *testing.T) {
	privateKeyPath := writeTestPrivateKey(t, makeEncryptedTestPrivateKeyPEM(t, "secret"))
	t.Setenv("DBEE_TEST_SNOWFLAKE_KEY_PASSWORD", "secret")

	cfg, err := prepareSnowflakeConfig("user@account/db/schema?privateKeyPath=" + privateKeyPath + "&privateKeyPassphraseEnv=DBEE_TEST_SNOWFLAKE_KEY_PASSWORD")
	require.NoError(t, err)

	assert.Equal(t, gosnowflake.AuthTypeJwt, cfg.Authenticator)
	assert.NotNil(t, cfg.PrivateKey)
}

func TestPrepareSnowflakeConfigDelegatesOAuthClientCredentials(t *testing.T) {
	cfg, err := prepareSnowflakeConfig("user@account/db/schema?authenticator=oauth_client_credentials&oauthClientId=id&oauthClientSecret=secret&oauthTokenRequestUrl=https%3A%2F%2Fexample.com%2Foauth%2Ftoken")
	require.NoError(t, err)

	assert.Equal(t, gosnowflake.AuthTypeOAuthClientCredentials, cfg.Authenticator)
	assert.Equal(t, "id", cfg.OauthClientID)
	assert.Equal(t, "secret", cfg.OauthClientSecret)
	assert.Equal(t, "https://example.com/oauth/token", cfg.OauthTokenRequestURL)
}

func TestSnowflakePingTimeoutAllowsInteractiveAuth(t *testing.T) {
	assert.Equal(t, 30*time.Second, snowflakePingTimeout(gosnowflake.Config{}))
	assert.Equal(t, 2*time.Minute, snowflakePingTimeout(gosnowflake.Config{Authenticator: gosnowflake.AuthTypeOAuthAuthorizationCode}))
	assert.Equal(t, 2*time.Minute, snowflakePingTimeout(gosnowflake.Config{Authenticator: gosnowflake.AuthTypeExternalBrowser}))
}

func TestSnowflakeIdentifier(t *testing.T) {
	assert.Equal(t, `"RAW"`, snowflakeIdentifier("RAW"))
	assert.Equal(t, `"SCHEMA""NAME"`, snowflakeIdentifier(`SCHEMA"NAME`))
	assert.Equal(t, `"DB"."SCHEMA"."TABLE"`, snowflakeTableName("DB", "SCHEMA", "TABLE"))
}

func TestSnowflakeDatabaseNameFromRowUsesNameHeader(t *testing.T) {
	got := snowflakeDatabaseNameFromRow([]string{"created_on", "kind", "name"}, []any{"date", "STANDARD", "DB"})

	assert.Equal(t, "DB", got)
}

func TestSnowflakeDatabaseNameFromRowFallsBackToShowDatabaseIndex(t *testing.T) {
	got := snowflakeDatabaseNameFromRow([]string{"header_0", "header_1"}, []any{"date", "DB"})

	assert.Equal(t, "DB", got)
}

func TestApplySnowflakeSessionDefaults(t *testing.T) {
	cfg := &gosnowflake.Config{}

	applySnowflakeSessionDefaults(cfg, sql.NullString{String: "DB", Valid: true}, sql.NullString{String: "SCHEMA", Valid: true})

	assert.Equal(t, "DB", cfg.Database)
	assert.Equal(t, "SCHEMA", cfg.Schema)
}

func TestApplySnowflakeSessionDefaultsDoesNotOverrideExplicitDSNValues(t *testing.T) {
	cfg := &gosnowflake.Config{Database: "EXPLICIT_DB", Schema: "EXPLICIT_SCHEMA"}

	applySnowflakeSessionDefaults(cfg, sql.NullString{String: "DB", Valid: true}, sql.NullString{String: "SCHEMA", Valid: true})

	assert.Equal(t, "EXPLICIT_DB", cfg.Database)
	assert.Equal(t, "EXPLICIT_SCHEMA", cfg.Schema)
}

func writeTestPrivateKey(t *testing.T, contents []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "snowflake_key.p8")
	require.NoError(t, os.WriteFile(path, contents, 0o600))
	return path
}

func makeTestPrivateKeyPEM(t *testing.T) []byte {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
}

func makeEncryptedTestPrivateKeyPEM(t *testing.T, passphrase string) []byte {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := pkcs8.ConvertPrivateKeyToPKCS8(key, []byte(passphrase))
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "ENCRYPTED PRIVATE KEY", Bytes: der})
}
