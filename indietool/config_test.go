package indietool

import (
	"testing"
)

func TestSecretsDirectoryCalculation(t *testing.T) {
	// Test default config
	t.Run("Default Config", func(t *testing.T) {
		defaultCfg := GetDefaultConfig()
		t.Logf("Config Path: %s", defaultCfg.Path)
		t.Logf("Initial Secrets Dir: %s", defaultCfg.Secrets.StorageDir)
		
		secretsCfg := defaultCfg.GetSecretsConfig()
		t.Logf("Final Secrets Dir: %s", secretsCfg.StorageDir)
		
		// Should be under the default config directory
		expected := "~/.config/indietool/secrets/default"
		if secretsCfg.StorageDir != expected {
			t.Errorf("Expected secrets dir %s, got %s", expected, secretsCfg.StorageDir)
		}
	})
	
	// Test custom config path
	t.Run("Custom Config Path", func(t *testing.T) {
		customCfg := GetDefaultConfig()
		customCfg.Path = "/custom/path/myapp.yaml"
		t.Logf("Config Path: %s", customCfg.Path)
		t.Logf("Initial Secrets Dir: %s", customCfg.Secrets.StorageDir)
		
		secretsCfg := customCfg.GetSecretsConfig()
		t.Logf("Final Secrets Dir: %s", secretsCfg.StorageDir)
		
		// Should be under the custom config directory
		expected := "/custom/path/secrets/default"
		if secretsCfg.StorageDir != expected {
			t.Errorf("Expected secrets dir %s, got %s", expected, secretsCfg.StorageDir)
		}
	})
	
	// Test when user has explicitly set a custom secrets directory
	t.Run("Custom Secrets Directory", func(t *testing.T) {
		customCfg := GetDefaultConfig()
		customCfg.Path = "/custom/path/myapp.yaml"
		customCfg.Secrets.StorageDir = "/completely/different/secrets"
		t.Logf("Config Path: %s", customCfg.Path)
		t.Logf("Initial Secrets Dir: %s", customCfg.Secrets.StorageDir)
		
		secretsCfg := customCfg.GetSecretsConfig()
		t.Logf("Final Secrets Dir: %s", secretsCfg.StorageDir)
		
		// Should keep the user's custom directory
		expected := "/completely/different/secrets"
		if secretsCfg.StorageDir != expected {
			t.Errorf("Expected secrets dir %s, got %s", expected, secretsCfg.StorageDir)
		}
	})
}