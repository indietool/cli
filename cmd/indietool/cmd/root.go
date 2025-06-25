/*
Copyright © 2025
*/
package cmd

import (
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"indietool/cli/config"
)

var (
	configPath string
	jsonOutput bool
	appConfig  *config.Config // Global config instance
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

	// Get home directory for default config path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %v", err)
	}
	defaultConfigPath := filepath.Join(homeDir, ".config", "indietool.yaml")

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", defaultConfigPath, "config file path")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output results in JSON format")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig loads the configuration from the specified config file path.
func initConfig() {
	// Load configuration using the new config package
	cfg, err := config.LoadConfigFromPath(configPath)
	if err != nil {
		// Don't fatal - just log the error and continue with empty config
		log.Warnf("Failed to load config from %s: %v", configPath, err)
		// Create empty config
		cfg = &config.Config{}
	}

	// Store the loaded (or empty) config globally
	appConfig = cfg

	// Only log success and validate if config is valid
	if cfg.Valid() {
		log.Infof("Loaded configuration from: %s", cfg.Path)

		// Optional: Validate the configuration
		if errors := cfg.ValidateConfig(); len(errors) > 0 {
			log.Warnf("Configuration validation warnings:")
			for _, errMsg := range errors {
				log.Warnf("  - %s", errMsg)
			}
		}
	} else {
		log.Warnf("No valid configuration loaded - using empty config")
	}
}

// GetConfig returns the globally loaded configuration instance.
// This function should be called from other commands to access the config.
func GetConfig() *config.Config {
	return appConfig
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
