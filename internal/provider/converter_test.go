package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/supermetal-inc/terraform-provider-supermetal/internal/api"
)

func TestPostgresSourceToAPI_Basic(t *testing.T) {
	model := &PostgresSourceModel{
		Host:     types.StringValue("localhost"),
		Port:     types.Int32Value(5432),
		Database: types.StringValue("testdb"),
		User:     types.StringValue("testuser"),
		Password: types.StringValue("testpass"),
		SslMode:  types.StringValue("Prefer"),
	}

	result, diags := model.toAPI()
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}

	if result.Connection.Host != "localhost" {
		t.Errorf("expected host localhost, got %s", result.Connection.Host)
	}
	if *result.Connection.Port != 5432 {
		t.Errorf("expected port 5432, got %d", *result.Connection.Port)
	}
	if result.Connection.Database != "testdb" {
		t.Errorf("expected database testdb, got %s", result.Connection.Database)
	}
	if result.Connection.User != "testuser" {
		t.Errorf("expected user testuser, got %s", result.Connection.User)
	}
	if result.Connection.Password != "testpass" {
		t.Errorf("expected password testpass, got %s", result.Connection.Password)
	}
}

func TestPostgresSourceToAPI_WithCatalogAndColumns(t *testing.T) {
	model := &PostgresSourceModel{
		Host:     types.StringValue("localhost"),
		Port:     types.Int32Value(5432),
		Database: types.StringValue("testdb"),
		User:     types.StringValue("testuser"),
		Password: types.StringValue("testpass"),
		SslMode:  types.StringValue("Disable"),
		Catalog: &CatalogModel{
			Name:          types.StringValue("testdb"),
			DefaultAction: types.StringValue("Exclude"),
			Schemas: map[string]*CatalogSchemaModel{
				"public": {
					Action: types.StringValue("Include"),
					Tables: map[string]*CatalogTableModel{
						"users": {
							Action: types.StringValue("Include"),
							Columns: map[string]*CatalogColumnModel{
								"id": {
									Action: types.StringValue("Include"),
								},
								"email": {
									Action: types.StringValue("Include"),
								},
								"password_hash": {
									Action: types.StringValue("Exclude"),
								},
							},
						},
					},
				},
			},
		},
	}

	apiResult, diags := model.toAPI()
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}

	if apiResult.Catalog == nil {
		t.Fatal("expected catalog to be set")
	}
	if len(apiResult.Catalog.Schemas) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(apiResult.Catalog.Schemas))
	}
	if len(apiResult.Catalog.Schemas[0].Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(apiResult.Catalog.Schemas[0].Tables))
	}

	columns := apiResult.Catalog.Schemas[0].Tables[0].Columns
	if columns == nil || len(*columns) != 3 {
		t.Fatalf("expected 3 columns, got %v", columns)
	}

	colMap := make(map[string]api.ConnectorCatalogColumn)
	for _, col := range *columns {
		colMap[col.Name] = col
	}

	if _, ok := colMap["id"]; !ok {
		t.Error("expected 'id' column")
	}
	if _, ok := colMap["email"]; !ok {
		t.Error("expected 'email' column")
	}
	pwCol, ok := colMap["password_hash"]
	if !ok {
		t.Error("expected 'password_hash' column")
	}
	action, _ := pwCol.Action.AsConnectorCatalogAction()
	if action != api.Exclude {
		t.Errorf("expected password_hash column action Exclude, got %v", action)
	}
}

func TestPostgresSourceMerge_WithColumns(t *testing.T) {
	state := &PostgresSourceModel{
		Host:     types.StringValue("localhost"),
		Port:     types.Int32Value(5432),
		Database: types.StringValue("testdb"),
		User:     types.StringValue("testuser"),
		Password: types.StringValue("secret"),
		SslMode:  types.StringValue("Disable"),
	}

	actionInclude := api.ConnectorCatalogTable_Action{}
	_ = actionInclude.FromConnectorCatalogAction(api.Include)

	colActionInclude := api.ConnectorCatalogColumn_Action{}
	_ = colActionInclude.FromConnectorCatalogAction(api.Include)
	colActionExclude := api.ConnectorCatalogColumn_Action{}
	_ = colActionExclude.FromConnectorCatalogAction(api.Exclude)

	apiResp := api.ConnectorPostgresPostgresSource{
		Catalog: &api.ConnectorCatalogCatalog{
			Name: "testdb",
			Schemas: []api.ConnectorCatalogSchema{
				{
					Name: "public",
					Tables: []api.ConnectorCatalogTable{
						{
							Name:   "users",
							Action: &actionInclude,
							Columns: &[]api.ConnectorCatalogColumn{
								{
									Name:       "id",
									PrimaryKey: ptrTo(true),
									Action:     &colActionInclude,
								},
								{
									Name:   "email",
									Action: &colActionInclude,
								},
								{
									Name:   "password_hash",
									Action: &colActionExclude,
								},
							},
						},
					},
				},
			},
		},
		Connection:      api.ConnectorPostgresPostgres{},
		ReplicationType: api.ConnectorPostgresPostgresReplicationType{},
	}

	model := &PostgresSourceModel{}
	model.mergeFromAPI(apiResp, state)

	if model.Catalog == nil {
		t.Fatal("expected catalog to be set")
	}
	if len(model.Catalog.Schemas) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(model.Catalog.Schemas))
	}
	schema, ok := model.Catalog.Schemas["public"]
	if !ok {
		t.Fatal("expected 'public' schema")
	}
	if len(schema.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(schema.Tables))
	}

	table, ok := schema.Tables["users"]
	if !ok {
		t.Fatal("expected 'users' table")
	}
	columns := table.Columns
	if len(columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(columns))
	}

	if _, ok := columns["id"]; !ok {
		t.Error("expected 'id' column")
	}
	pwCol, ok := columns["password_hash"]
	if !ok {
		t.Error("expected 'password_hash' column")
	}
	if pwCol.Action.ValueString() != "Exclude" {
		t.Errorf("expected password_hash column action Exclude, got %s", pwCol.Action.ValueString())
	}
}

func TestDuckDBSinkToAPI(t *testing.T) {
	model := &DuckDBSinkModel{
		TargetDatabase: types.StringValue("main"),
		TargetSchema:   types.StringValue("public"),
		Connection: &DuckdbConnectionModel{
			Quack: &DuckdbQuackModel{
				URL:   types.StringValue("http://localhost:9494"),
				Token: types.StringValue("secret-token"),
				Ssl:   types.BoolValue(false),
			},
		},
	}

	result, diags := model.toAPI()
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}

	if result.TargetDatabase != "main" {
		t.Errorf("expected target_database main, got %s", result.TargetDatabase)
	}
	if *result.TargetSchema != "public" {
		t.Errorf("expected target_schema public, got %s", *result.TargetSchema)
	}

	conn, err := result.Connection.AsConnectorDuckdbDuckDbConnection0()
	if err != nil {
		t.Fatalf("expected quack connection: %v", err)
	}
	if conn.Quack.Url != "http://localhost:9494" {
		t.Errorf("expected url http://localhost:9494, got %s", conn.Quack.Url)
	}
	if *conn.Quack.Token != "secret-token" {
		t.Errorf("expected token secret-token, got %s", *conn.Quack.Token)
	}
	if *conn.Quack.Ssl != false {
		t.Error("expected ssl to be false")
	}
}

func TestSnowflakeSinkToAPI_PasswordAuth(t *testing.T) {
	model := &SnowflakeSinkModel{
		AccountIdentifier: types.StringValue("myorg-account123"),
		Warehouse:         types.StringValue("COMPUTE_WH"),
		User:              types.StringValue("datauser"),
		Role:              types.StringValue("DATA_ROLE"),
		TargetDatabase:    types.StringValue("ANALYTICS"),
		TargetSchema:      types.StringValue("RAW"),
		UseTransactions:   types.BoolValue(true),
		Auth: &SnowflakeAuthModel{
			Password: &SnowflakePasswordModel{
				Password: types.StringValue("s3cr3t"),
			},
		},
	}

	result, diags := model.toAPI()
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}

	if result.Connection.AccountIdentifier != "myorg-account123" {
		t.Errorf("expected account_identifier myorg-account123, got %s", result.Connection.AccountIdentifier)
	}
	if result.Connection.Warehouse != "COMPUTE_WH" {
		t.Errorf("expected warehouse COMPUTE_WH, got %s", result.Connection.Warehouse)
	}
	if result.Connection.User != "datauser" {
		t.Errorf("expected user datauser, got %s", result.Connection.User)
	}
	if *result.Connection.Role != "DATA_ROLE" {
		t.Errorf("expected role DATA_ROLE, got %s", *result.Connection.Role)
	}
	if result.TargetDatabase != "ANALYTICS" {
		t.Errorf("expected target_database ANALYTICS, got %s", result.TargetDatabase)
	}
	if *result.TargetSchema != "RAW" {
		t.Errorf("expected target_schema RAW, got %s", *result.TargetSchema)
	}
	if *result.UseTransactions != true {
		t.Error("expected use_transactions to be true")
	}

	auth, err := result.Connection.Auth.AsConnectorSnowflakeAuth0()
	if err != nil {
		t.Fatalf("expected password auth: %v", err)
	}
	if auth.Password.Password != "s3cr3t" {
		t.Errorf("expected password s3cr3t, got %s", auth.Password.Password)
	}
}

func TestSnowflakeSinkToAPI_KeyPairAuth(t *testing.T) {
	model := &SnowflakeSinkModel{
		AccountIdentifier: types.StringValue("myorg-account123"),
		Warehouse:         types.StringValue("COMPUTE_WH"),
		User:              types.StringValue("datauser"),
		TargetDatabase:    types.StringValue("ANALYTICS"),
		UseTransactions:   types.BoolValue(false),
		Auth: &SnowflakeAuthModel{
			KeyPair: &SnowflakeKeyPairModel{
				PrivateKeyPEM:      types.StringValue("-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----"),
				PrivateKeyPassword: types.StringValue("keypass"),
			},
		},
	}

	result, diags := model.toAPI()
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}

	auth, err := result.Connection.Auth.AsConnectorSnowflakeAuth1()
	if err != nil {
		t.Fatalf("expected key pair auth: %v", err)
	}
	if auth.KeyPair.PrivateKeyPem != "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----" {
		t.Errorf("unexpected private_key_pem: %s", auth.KeyPair.PrivateKeyPem)
	}
	if *auth.KeyPair.PrivateKeyPassword != "keypass" {
		t.Errorf("expected private_key_password keypass, got %s", *auth.KeyPair.PrivateKeyPassword)
	}
}

func TestPostgresSourceMerge_PasswordPreserved(t *testing.T) {
	state := &PostgresSourceModel{
		Host:     types.StringValue("dbhost"),
		Port:     types.Int32Value(5432),
		Database: types.StringValue("testdb"),
		User:     types.StringValue("user"),
		Password: types.StringValue("secret-password"),
		SslMode:  types.StringValue("Disable"),
	}

	port := int32(5432)
	apiResp := api.ConnectorPostgresPostgresSource{
		Connection: api.ConnectorPostgresPostgres{
			Host:     "dbhost",
			Port:     &port,
			Database: "testdb",
			User:     "user",
			Password: "", // scrubbed
		},
	}

	model := &PostgresSourceModel{}
	model.mergeFromAPI(apiResp, state)

	if model.Password.ValueString() != "secret-password" {
		t.Errorf("expected password to be preserved from state, got %q", model.Password.ValueString())
	}
}

func TestSnowflakeMerge_PasswordPreserved(t *testing.T) {
	state := &SnowflakeSinkModel{
		AccountIdentifier: types.StringValue("myorg-account"),
		Warehouse:         types.StringValue("WH"),
		User:              types.StringValue("user"),
		TargetDatabase:    types.StringValue("DB"),
		Auth: &SnowflakeAuthModel{
			Password: &SnowflakePasswordModel{
				Password: types.StringValue("secret-password"),
			},
		},
	}

	apiResp := api.ConnectorSnowflakeSnowflakeSink{
		Connection: api.ConnectorSnowflakeSnowflake{
			AccountIdentifier: "myorg-account",
			Warehouse:         "WH",
			User:              "user",
			Auth:              api.ConnectorSnowflakeAuth{},
		},
		TargetDatabase: "DB",
	}
	_ = apiResp.Connection.Auth.FromConnectorSnowflakeAuth0(api.ConnectorSnowflakeAuth0{
		Password: api.ConnectorSnowflakePassword{
			Password: "", // scrubbed
		},
	})

	model := &SnowflakeSinkModel{}
	model.mergeFromAPI(apiResp, state)

	if model.Auth == nil || model.Auth.Password == nil {
		t.Fatal("expected auth.password to be populated")
	}
	if model.Auth.Password.Password.ValueString() != "secret-password" {
		t.Errorf("expected password to be preserved from state, got %q", model.Auth.Password.Password.ValueString())
	}
}

func TestSnowflakeMerge_KeyPairSecretsPreserved(t *testing.T) {
	state := &SnowflakeSinkModel{
		AccountIdentifier: types.StringValue("myorg-account"),
		Warehouse:         types.StringValue("WH"),
		User:              types.StringValue("user"),
		TargetDatabase:    types.StringValue("DB"),
		Auth: &SnowflakeAuthModel{
			KeyPair: &SnowflakeKeyPairModel{
				PrivateKeyPEM:      types.StringValue("-----BEGIN PRIVATE KEY-----\nsecret\n-----END PRIVATE KEY-----"),
				PrivateKeyPassword: types.StringValue("keypass"),
			},
		},
	}

	apiResp := api.ConnectorSnowflakeSnowflakeSink{
		Connection: api.ConnectorSnowflakeSnowflake{
			AccountIdentifier: "myorg-account",
			Warehouse:         "WH",
			User:              "user",
			Auth:              api.ConnectorSnowflakeAuth{},
		},
		TargetDatabase: "DB",
	}
	_ = apiResp.Connection.Auth.FromConnectorSnowflakeAuth1(api.ConnectorSnowflakeAuth1{
		KeyPair: api.ConnectorSnowflakeKeyPair{
			PrivateKeyPem:      "",        // scrubbed
			PrivateKeyPassword: ptrTo(""), // scrubbed
		},
	})

	model := &SnowflakeSinkModel{}
	model.mergeFromAPI(apiResp, state)

	if model.Auth == nil || model.Auth.KeyPair == nil {
		t.Fatal("expected auth.key_pair to be populated")
	}
	if model.Auth.KeyPair.PrivateKeyPEM.ValueString() != "-----BEGIN PRIVATE KEY-----\nsecret\n-----END PRIVATE KEY-----" {
		t.Errorf("expected private_key_pem to be preserved from state, got %q", model.Auth.KeyPair.PrivateKeyPEM.ValueString())
	}
	if model.Auth.KeyPair.PrivateKeyPassword.ValueString() != "keypass" {
		t.Errorf("expected private_key_password to be preserved from state, got %q", model.Auth.KeyPair.PrivateKeyPassword.ValueString())
	}
}

func TestPostgresSourceMerge_SshTunnelSecretPreserved(t *testing.T) {
	port := int32(22)
	state := &PostgresSourceModel{
		Host:     types.StringValue("dbhost"),
		Port:     types.Int32Value(5432),
		Database: types.StringValue("testdb"),
		User:     types.StringValue("user"),
		Password: types.StringValue("dbpass"),
		Tunnel: &SshTunnelTypeModel{
			Ssh: &SshTunnelModel{
				BastionHost: types.StringValue("bastion.example.com"),
				BastionPort: types.Int32Value(22),
				User:        types.StringValue("sshuser"),
				Auth: &SshAuthModel{
					BringYourOwnKey: &SshBringYourOwnKeyModel{
						PrivateKey: types.StringValue("-----BEGIN OPENSSH PRIVATE KEY-----\nsecret\n-----END OPENSSH PRIVATE KEY-----"),
					},
				},
			},
		},
	}

	apiResp := api.ConnectorPostgresPostgresSource{
		Connection: api.ConnectorPostgresPostgres{
			Host:     "dbhost",
			Port:     ptrTo(int32(5432)),
			Database: "testdb",
			User:     "user",
			Password: "",
			Tunnel:   &api.ConnectorSshTunnelType{},
		},
		ReplicationType: api.ConnectorPostgresPostgresReplicationType{},
	}
	_ = apiResp.Connection.Tunnel.FromConnectorSshTunnelType0(api.ConnectorSshTunnelType0{
		Ssh: api.ConnectorSshSshTunnel{
			BastionHost: "bastion.example.com",
			BastionPort: &port,
			User:        "sshuser",
			Auth:        api.ConnectorSshSshAuth{},
		},
	})
	sshTunnel, _ := apiResp.Connection.Tunnel.AsConnectorSshTunnelType0()
	_ = sshTunnel.Ssh.Auth.FromConnectorSshSshAuth1(api.ConnectorSshSshAuth1{
		BringYourOwnKey: api.ConnectorSshBringYourOwnKey{
			PrivateKey: "", // scrubbed
		},
	})
	_ = apiResp.Connection.Tunnel.FromConnectorSshTunnelType0(sshTunnel)

	model := &PostgresSourceModel{}
	model.mergeFromAPI(apiResp, state)

	if model.Tunnel == nil || model.Tunnel.Ssh == nil || model.Tunnel.Ssh.Auth == nil || model.Tunnel.Ssh.Auth.BringYourOwnKey == nil {
		t.Fatal("expected tunnel.ssh.auth.bring_your_own_key to be populated")
	}
	if model.Tunnel.Ssh.Auth.BringYourOwnKey.PrivateKey.ValueString() != "-----BEGIN OPENSSH PRIVATE KEY-----\nsecret\n-----END OPENSSH PRIVATE KEY-----" {
		t.Errorf("expected private_key to be preserved from state, got %q", model.Tunnel.Ssh.Auth.BringYourOwnKey.PrivateKey.ValueString())
	}
}

func ptrTo[T any](v T) *T {
	return &v
}
