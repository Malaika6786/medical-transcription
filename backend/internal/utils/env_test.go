package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigProcessEnvironmentOverridesDotEnv(t *testing.T) {
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(workingDir) })

	temporaryDir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(temporaryDir, ".env"),
		[]byte("AI_MODEL=file-model\nAI_API_KEY=file-key\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(temporaryDir); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AI_MODEL", "process-model")
	t.Setenv("AI_API_KEY", "process-key")

	config := LoadConfig()
	if config.AIModel != "process-model" {
		t.Fatalf("AI_MODEL = %q, want process-model", config.AIModel)
	}
	if config.AIAPIKey != "process-key" {
		t.Fatalf("AI_API_KEY = %q, want process-key", config.AIAPIKey)
	}
}
