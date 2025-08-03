package cmd

import (
	"indietool/cli/indietool"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

// expandTildePath expands ~ to the user's home directory in a file path
func expandTildePath(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return path // Return original path if we can't get home dir
	}

	return filepath.Join(homeDir, path[2:])
}

var (
	version = "dev"
	// configPath        string
	// defaultConfigPath string // Store default config path to detect when using default
	jsonOutput       bool
	providerRegistry *indietool.Registry // Global provider registry

	appConfig = indietool.GetDefaultConfig() // Get a copy of default config
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "indietool",
	Short: "indie builder toolkit",
	//	Long: `A longer description that spans multiple lines and likely contains
	//
	// examples and usage of using your application. For example:
	//
	// Cobra is a CLI library for Go that empowers applications.
	// This application is a tool to generate the needed files
	// to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		saveConfigIfValid()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVarP(&appConfig.Path, "config", "c", appConfig.Path, "config file path")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output results in JSON format")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig loads the configuration from the specified config file path.
func initConfig() {
	// Expand tilde in the config path before loading
	expandedConfigPath := expandTildePath(appConfig.Path)

	// Load configuration using the expanded path
	cfg, err := indietool.LoadFromPath(expandedConfigPath)
	if err != nil {
		expandedDefaultPath := expandTildePath(indietool.DefaultConfigFileLocation)

		// Check if we're using the default config path and the file doesn't exist
		if expandedConfigPath == expandedDefaultPath && os.IsNotExist(err) {
			log.Infof("No config file found at default location, creating default config at: %s", expandedDefaultPath)

			// Create default config
			cfg := indietool.GetDefaultConfig()

			// Ensure the config directory exists (with all parent directories)
			configDir := filepath.Dir(expandedDefaultPath)
			if err := os.MkdirAll(configDir, 0755); err != nil {
				log.Warnf("Failed to create config directory %s: %v", configDir, err)
			} else {
				// Save the default config to the expanded location
				if err := cfg.SaveConfig(expandedDefaultPath); err != nil {
					log.Warnf("Failed to save default config to %s: %v", expandedDefaultPath, err)
				} else {
					// Set the path so the config becomes "valid"
					cfg.Path = expandedDefaultPath
					log.Infof("Created default configuration file at: %s", expandedDefaultPath)
				}
			}
		} else {
			// For other errors (non-default path, file exists but corrupted, etc.)
			log.Warnf("Failed to load config from %s: %v", expandedConfigPath, err)
			// Create default config without saving
			cfg = indietool.GetDefaultConfig()
		}
	}

	// Store the loaded (or empty) config globally
	appConfig = cfg
	appConfig.Version = version

	// Only log success and validate if config is valid
	if cfg.Valid() {
		log.Debugf("Loaded configuration from: %s", cfg.Path)

		// Optional: Validate the configuration
		if errors := cfg.ValidateConfig(); len(errors) > 0 {
			log.Warnf("Configuration validation warnings:")
			for _, errMsg := range errors {
				log.Warnf("  - %s", errMsg)
			}
		}

		// Initialize provider registry with configured providers
		initProviderRegistry(cfg)
	} else {
		log.Warnf("No valid configuration loaded - using empty config")
		// Initialize empty registry
		registry, _ := indietool.NewRegistry(&indietool.Config{})
		providerRegistry = registry
	}
}

// initProviderRegistry creates and configures the global provider registry
// based on the loaded configuration. Only called when config is valid.
func initProviderRegistry(cfg *indietool.Config) {
	registry, err := indietool.NewRegistry(cfg)
	if err != nil {
		log.Warnf("Failed to create provider registry: %v", err)
		// Create empty registry as fallback
		registry, _ = indietool.NewRegistry(&indietool.Config{})
	}
	providerRegistry = registry

	// Log summary of configured providers
	enabledCount := 0
	configuredCount := 0

	if cfg.Providers.Cloudflare != nil {
		configuredCount++
		if cfg.Providers.Cloudflare.Enabled {
			enabledCount++
		}
	}
	if cfg.Providers.Porkbun != nil {
		configuredCount++
		if cfg.Providers.Porkbun.Enabled {
			enabledCount++
		}
	}
	if cfg.Providers.Namecheap != nil {
		configuredCount++
		if cfg.Providers.Namecheap.Enabled {
			enabledCount++
		}
	}
	if cfg.Providers.GoDaddy != nil {
		configuredCount++
		if cfg.Providers.GoDaddy.Enabled {
			enabledCount++
		}
	}

	if configuredCount > 0 {
		log.Debugf("Configured %d provider(s)", configuredCount)
		log.Debugf("Enabled %d provider(s)", enabledCount)
	} else {
		log.Debugf("No providers configured")
	}
}

// GetConfig returns the globally loaded configuration instance.
// This function should be called from other commands to access the config.
func GetConfig() *indietool.Config {
	return appConfig
}

// GetProviderRegistry returns the globally initialized provider registry.
// This function should be called from other commands to access providers.
func GetProviderRegistry() *indietool.Registry {
	return providerRegistry
}

// SetVersion sets the version in the global app config
func SetVersion(appVersion string) {
	if appConfig != nil {
		appConfig.Version = appVersion
	}

	if metricsAgent != nil {
		metricsAgent.SetVersion(appVersion)
	}

	version = appVersion
}

// GetVersion returns the version from the global app config
func GetVersion() string {
	if appConfig != nil {
		return appConfig.Version
	}
	return "dev"
}

// saveConfigIfValid saves the configuration back to its loaded path if it's valid.
// This function is called in the PersistentPostRun hook to persist any changes.
func saveConfigIfValid() {
	if appConfig == nil {
		return // No config to save
	}

	if !appConfig.Valid() {
		return // Config not valid, nothing to save
	}

	// Don't save if loaded from Viper (not a real file path)
	if appConfig.Path == "<viper>" {
		log.Debugf("Skipping save - config was loaded from Viper, not a file")
		return
	}

	// Save the config back to the path it was loaded from
	err := appConfig.SaveConfig(appConfig.Path)
	if err != nil {
		// Don't crash on save errors, just log them
		log.Warnf("Failed to save config to %s: %v", appConfig.Path, err)
	} else {
		log.Debugf("Saved configuration to: %s", appConfig.Path)
	}
}
