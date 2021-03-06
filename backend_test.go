package boundarysecrets

import (
	"context"
	"github.com/hashicorp/boundary/api/accounts"
	"github.com/hashicorp/boundary/api/users"
	"os"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/require"
)

const (
	envVarRunAccTests       = "VAULT_ACC"
	envVarBoundaryLoginName = "TEST_BOUNDARY_LOGIN_NAME"
	envVarBoundaryPassword  = "TEST_BOUNDARY_PASSWORD"
	envVarBoundaryAddr      = "TEST_BOUNDARY_ADDR"
)

// getTestBackend will help you construct a test backend object.
// Update this function with your target backend.
func getTestBackend(tb testing.TB) (*boundaryBackend, logical.Storage) {
	tb.Helper()

	config := logical.TestBackendConfig()
	config.StorageView = new(logical.InmemStorage)
	config.Logger = hclog.NewNullLogger()
	config.System = logical.TestSystemView()

	b, err := Factory(context.Background(), config)
	if err != nil {
		tb.Fatal(err)
	}

	return b.(*boundaryBackend), config.StorageView
}

// runAcceptanceTests will separate unit tests from
// acceptance tests, which will make active requests
// to your target API.
var runAcceptanceTests = os.Getenv(envVarRunAccTests) == "1"

// testEnv creates an object to store and track testing environment
// resources
type testEnv struct {
	LoginName    string
	Password     string
	Addr         string
	AuthMethodId string

	Backend logical.Backend
	Context context.Context
	Storage logical.Storage

	// SecretToken tracks the API token, for checking rotations
	UserId string

	// Tokens tracks the generated tokens, to make sure we clean up
	AccountId string
}

// AddConfig adds the configuration to the test backend.
// Make sure data includes all of the configuration
// attributes you need and the `config` path!
func (e *testEnv) AddConfig(t *testing.T) {
	req := &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "config",
		Storage:   e.Storage,
		Data: map[string]interface{}{
			"login_name":     e.LoginName,
			"password":       e.Password,
			"addr":           e.Addr,
			"auth_method_id": e.AuthMethodId,
		},
	}
	resp, err := e.Backend.HandleRequest(e.Context, req)
	require.Nil(t, resp)
	require.Nil(t, err)
}

// AddUserTokenRole adds a role for the HashiCups
// user token.
func (e *testEnv) AddUserTokenRole(t *testing.T) {
	req := &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "role/test-user-token",
		Storage:   e.Storage,
		Data: map[string]interface{}{
			"login_name": e.LoginName,
		},
	}
	resp, err := e.Backend.HandleRequest(e.Context, req)
	require.Nil(t, resp)
	require.Nil(t, err)
}

// ReadUserToken retrieves the user token
// based on a Vault role.
func (e *testEnv) ReadUserToken(t *testing.T) {
	req := &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "creds/test-user-token",
		Storage:   e.Storage,
	}
	resp, err := e.Backend.HandleRequest(e.Context, req)
	require.Nil(t, err)
	require.NotNil(t, resp)

	if t, ok := resp.Data["password"]; ok {
		e.Password = t.(string)
	}
	require.NotEmpty(t, resp.Data["password"])

	if t, ok := resp.Data["login_name"]; ok {
		e.LoginName = t.(string)
	}
	require.NotEmpty(t, resp.Data["login_name"])

	if t, ok := resp.Data["auth_method_id"]; ok {
		e.AuthMethodId = t.(string)
	}
	require.NotEmpty(t, resp.Data["auth_method_id"])

	if t, ok := resp.Data["account_id"]; ok {
		e.AccountId = t.(string)
	}
	require.NotEmpty(t, resp.Data["account_id"])

	if t, ok := resp.Data["user_id"]; ok {
		e.UserId = t.(string)
	}
	require.NotEmpty(t, resp.Data["user_id"])

	//if e.SecretToken != "" {
	//	require.NotEqual(t, e.SecretToken, resp.Data["token"])
	//}
	//
	//// collect secret IDs to revoke at end of test
	//require.NotNil(t, resp.Secret)
	//if t, ok := resp.Secret.InternalData["token"]; ok {
	//	e.SecretToken = t.(string)
	//}
}

// CleanupUserTokens removes the tokens
// when the test completes.
func (e *testEnv) CleanupUserTokens(t *testing.T) {
	//if len(e.Tokens) == 0 {
	//	t.Fatalf("expected 2 tokens, got: %d", len(e.Tokens))
	//}
	//
	//for _, token := range e.Tokens {
	//	b := e.Backend.(*boundaryBackend)
	//	client, err := b.getClient(e.Context, e.Storage)
	//	if err != nil {
	//		t.Fatal("fatal getting client")
	//	}
	//	client.Client.Token = string(token)
	//	if err := client.SignOut(); err != nil {
	//		t.Fatalf("unexpected error deleting user token: %s", err)
	//	}
	//}

	b := e.Backend.(*boundaryBackend)
	client, err := b.getClient(e.Context, e.Storage)
	if err != nil {
		t.Fatal("fatal getting client")
	}

	ucr := users.NewClient(client.Client)
	var userOpts []users.Option
	_, err = ucr.Delete(e.Context, e.UserId, userOpts...)
	if err != nil {
		t.Fatalf("unexpected error deleting user: %s", err)
	}

	acr := accounts.NewClient(client.Client)

	var opts []accounts.Option
	_, err = acr.Delete(e.Context, e.AccountId, opts...)
	if err != nil {
		t.Fatalf("unexpected error deleting account: %s", err)
	}

}
