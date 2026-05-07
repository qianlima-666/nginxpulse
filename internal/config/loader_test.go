package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigExpandsEnvPlaceholdersFromConfigFile(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "configs")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}

	configJSON := `{
  "websites": [
    {
      "name": "sftp-site",
      "sources": [
        {
          "id": "sftp-main",
          "type": "sftp",
          "host": "127.0.0.1",
          "user": "root",
          "auth": {
            "password": "${NGINXPULSE_SFTP_PASSWORD}"
          },
          "path": "/var/log/nginx/access.log"
        }
      ]
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(configDir, "nginxpulse_config.json"), []byte(configJSON), 0o644); err != nil {
		t.Fatalf("WriteFile config error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, ".env"), []byte("NGINXPULSE_SFTP_PASSWORD=from-dotenv\n"), 0o644); err != nil {
		t.Fatalf("WriteFile .env error: %v", err)
	}

	restoreCwd := mustChdir(t, tempDir)
	defer restoreCwd()

	unsetEnv(t, envConfigJSON)
	unsetEnv(t, envWebsites)
	unsetEnv(t, "NGINXPULSE_SFTP_PASSWORD")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}

	got := cfg.Websites[0].Sources[0].Auth.Password
	if got != "from-dotenv" {
		t.Fatalf("password = %q", got)
	}
}

func TestLoadConfigExpandsEnvPlaceholdersFromConfigJSON(t *testing.T) {
	unsetEnv(t, envWebsites)
	t.Setenv("NGINXPULSE_SFTP_PASSWORD", "from-env")
	t.Setenv(envConfigJSON, `{
  "websites": [
    {
      "name": "sftp-site",
      "sources": [
        {
          "id": "sftp-main",
          "type": "sftp",
          "host": "127.0.0.1",
          "user": "root",
          "auth": {
            "password": "${NGINXPULSE_SFTP_PASSWORD}"
          },
          "path": "/var/log/nginx/access.log"
        }
      ]
    }
  ]
}`)

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}

	got := cfg.Websites[0].Sources[0].Auth.Password
	if got != "from-env" {
		t.Fatalf("password = %q", got)
	}
}

func TestLoadConfigFailsWhenPlaceholderEnvMissing(t *testing.T) {
	unsetEnv(t, envWebsites)
	unsetEnv(t, "NGINXPULSE_SFTP_PASSWORD")
	t.Setenv(envConfigJSON, `{
  "websites": [
    {
      "name": "sftp-site",
      "sources": [
        {
          "id": "sftp-main",
          "type": "sftp",
          "host": "127.0.0.1",
          "user": "root",
          "auth": {
            "password": "${NGINXPULSE_SFTP_PASSWORD}"
          },
          "path": "/var/log/nginx/access.log"
        }
      ]
    }
  ]
}`)

	if _, err := loadConfig(); err == nil {
		t.Fatal("expected loadConfig to fail when env placeholder is missing")
	}
}

func mustChdir(t *testing.T, dir string) func() {
	t.Helper()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd error: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir error: %v", err)
	}
	return func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore cwd error: %v", err)
		}
	}
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()

	oldValue, existed := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("Unsetenv(%s) error: %v", key, err)
	}
	t.Cleanup(func() {
		var err error
		if existed {
			err = os.Setenv(key, oldValue)
		} else {
			err = os.Unsetenv(key)
		}
		if err != nil {
			t.Fatalf("restore env %s error: %v", key, err)
		}
	})
}
