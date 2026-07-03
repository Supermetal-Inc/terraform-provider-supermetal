package provider_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/supermetal-inc/terraform-provider-supermetal/internal/provider"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func getAgentBinaryPath() string {
	return os.Getenv("SUPERMETAL_AGENT_BINARY")
}

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"supermetal": providerserver.NewProtocol6WithError(provider.New("test")()),
}

type configOpt func(*configParams)

type configParams struct {
	endpoint       string
	host           string
	port           int
	id             string
	name           string
	resourceName   string
	database       string
	user           string
	password       string
	passwordExpr   string
	sslMode        string
	disabled       bool
	skipValidation bool
	catalog        string
	extra          string
}

func withPassword(pw string) configOpt {
	return func(p *configParams) { p.password = pw }
}

func withPasswordExpr(expr string) configOpt {
	return func(p *configParams) { p.passwordExpr = expr }
}

func withDisabled() configOpt {
	return func(p *configParams) { p.disabled = true }
}

func withCatalog(hcl string) configOpt {
	return func(p *configParams) { p.catalog = hcl }
}

func withValidation() configOpt {
	return func(p *configParams) { p.skipValidation = false }
}

func withExtra(hcl string) configOpt {
	return func(p *configParams) { p.extra = hcl }
}

func withResourceName(name string) configOpt {
	return func(p *configParams) { p.resourceName = name }
}

func withDatabase(db string) configOpt {
	return func(p *configParams) { p.database = db }
}

func (h *testHarness) config(id, name string, opts ...configOpt) string {
	h.t.Helper()
	p := &configParams{
		endpoint:       h.agentEndpoint,
		host:           h.postgresHost,
		port:           h.postgresPort,
		id:             id,
		name:           name,
		resourceName:   "test",
		database:       "testdb",
		user:           "testuser",
		password:       "testpass",
		sslMode:        "Disable",
		skipValidation: true,
	}
	for _, o := range opts {
		o(p)
	}
	return p.build()
}

func (p *configParams) build() string {
	var b strings.Builder

	fmt.Fprintf(&b, `
provider "supermetal" {
  endpoint        = %q
  skip_validation = %t
}
`, p.endpoint, p.skipValidation)

	if p.extra != "" {
		fmt.Fprintf(&b, "\n%s\n", p.extra)
	}

	disabledLine := ""
	if p.disabled {
		disabledLine = "\n  disabled = true"
	}

	passwordVal := fmt.Sprintf("%q", p.password)
	if p.passwordExpr != "" {
		passwordVal = p.passwordExpr
	}

	catalogBlock := ""
	if p.catalog != "" {
		catalogBlock = fmt.Sprintf("\n\n      catalog = {\n%s\n      }", p.catalog)
	}

	fmt.Fprintf(&b, `
resource "supermetal_connector" %q {
  id   = %q
  name = %q%s

  source = {
    postgres = {
      host     = %q
      port     = %d
      database = %q
      user     = %q
      password = %s
      ssl_mode = %q

      replication_type = {
        snapshot = {}
      }%s
    }
  }

  sink = {
    duckdb = {
      target_database = "main"
      connection = {
        quack = {
          url = "http://localhost:9494"
          ssl = false
        }
      }
    }
  }
}
`, p.resourceName, p.id, p.name, disabledLine,
		p.host, p.port, p.database, p.user, passwordVal, p.sslMode,
		catalogBlock)

	return b.String()
}

type testHarness struct {
	t             *testing.T
	agentCmd      *exec.Cmd
	agentEndpoint string
	postgresHost  string
	postgresPort  int
	tempDir       string
	postgresC     *postgres.PostgresContainer
}

func newTestHarness(t *testing.T) *testHarness {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless TF_ACC is set")
	}

	agentBinaryPath := getAgentBinaryPath()
	if agentBinaryPath == "" {
		t.Fatal("SUPERMETAL_AGENT_BINARY not set. Set it to the path of the supermetal agent binary.")
	}
	if _, err := os.Stat(agentBinaryPath); os.IsNotExist(err) {
		t.Fatalf("Agent binary not found at %s", agentBinaryPath)
	}

	ctx := context.Background()

	postgresC, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		postgres.BasicWaitStrategies(),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("Failed to start postgres container: %v", err)
	}

	host, err := postgresC.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get postgres host: %v", err)
	}
	mappedPort, err := postgresC.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("Failed to get postgres port: %v", err)
	}

	tempDir, err := os.MkdirTemp("", "supermetal-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	bufferDir := filepath.Join(tempDir, "buffer")
	if err := os.MkdirAll(bufferDir, 0755); err != nil {
		t.Fatalf("Failed to create buffer dir: %v", err)
	}

	agentPort, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to get free port: %v", err)
	}

	h := &testHarness{
		t:             t,
		agentEndpoint: fmt.Sprintf("http://localhost:%d", agentPort),
		postgresHost:  host,
		postgresPort:  int(mappedPort.Num()),
		tempDir:       tempDir,
		postgresC:     postgresC,
	}

	configPath := filepath.Join(tempDir, "config.json")
	configContent := fmt.Sprintf(`{
  "connectors": [
    {
      "id": "placeholder",
      "disabled": true,
      "buffer": {
        "object_store": {
          "url": "file://%s"
        }
      },
      "source": {
        "postgres": {
          "connection": {
            "host": "%s",
            "port": %d,
            "database": "testdb",
            "user": "testuser",
            "password": "testpass",
            "ssl_mode": "Disable"
          },
          "replication_type": {
            "snapshot": {}
          }
        }
      },
      "sink": {
        "postgres": {
          "connection": {
            "host": "%s",
            "port": %d,
            "database": "testdb",
            "user": "testuser",
            "password": "testpass",
            "ssl_mode": "Disable"
          }
        }
      }
    }
  ]
}`, bufferDir, host, mappedPort.Num(), host, mappedPort.Num())
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	t.Logf("Config written to %s:\n%s", configPath, configContent)

	h.agentCmd = exec.Command(getAgentBinaryPath(),
		"--config", configPath,
		"--server",
		"--server-port", fmt.Sprintf("%d", agentPort),
		"--no-prompt",
	)
	h.agentCmd.Stdout = os.Stdout
	h.agentCmd.Stderr = os.Stderr
	h.agentCmd.Env = append(os.Environ(),
		"XDG_DATA_HOME="+tempDir,
		"XDG_CONFIG_HOME="+tempDir,
	)

	if err := h.agentCmd.Start(); err != nil {
		t.Fatalf("Failed to start agent: %v", err)
	}

	if err := waitForAgent(h.agentEndpoint, 30*time.Second); err != nil {
		_ = h.agentCmd.Process.Kill()
		t.Fatalf("Agent did not become ready: %v", err)
	}

	t.Cleanup(h.cleanup)

	return h
}

func (h *testHarness) cleanup() {
	if h.agentCmd != nil && h.agentCmd.Process != nil {
		_ = h.agentCmd.Process.Kill()
		_ = h.agentCmd.Wait()
	}
	if h.postgresC != nil {
		_ = h.postgresC.Terminate(context.Background())
	}
	if h.tempDir != "" {
		_ = os.RemoveAll(h.tempDir)
	}
}

func getFreePort() (int, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func waitForAgent(endpoint string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(endpoint + "/api/v1/connectors")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("agent did not respond within %v", timeout)
}
