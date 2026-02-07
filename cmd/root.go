package cmd

import (
	"github.com/wsl-images/wslb/internal/app"
	"github.com/wsl-images/wslb/internal/logger"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile   string
	jsonOut   bool
	ndjsonOut bool
	services  *app.Services
)

var rootCmd = &cobra.Command{
	Use:          "wslb",
	Short:        "Build and manage WSL workspace images",
	SilenceUsage: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logger.Error("Error executing command: ", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.wslb/wslb.yaml)")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "machine-readable JSON output")
	rootCmd.PersistentFlags().BoolVar(&ndjsonOut, "ndjson", false, "stream NDJSON progress events")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			logger.Fatal("Unable to determine home directory: ", err)
		}

		configDir := filepath.Join(home, ".wslb")
		if err := os.MkdirAll(configDir, 0o755); err != nil {
			logger.Fatal("Unable to create config directory: ", err)
		}

		viper.AddConfigPath(configDir)
		viper.SetConfigName("wslb")
	}

	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err == nil {
		logger.Info("Using config file: ", viper.ConfigFileUsed())
	}
}

func getServices() *app.Services {
	if services == nil {
		services = app.NewServices()
	}
	return services
}
