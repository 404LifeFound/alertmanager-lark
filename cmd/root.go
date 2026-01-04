/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/404LifeFound/alertmanager-lark/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "alertmanager-lark",
		Aliases: []string{"alert", "lark-webhook"},
		Short:   "A webhook application",
		Long:    "The application support alertmanager send alert as card message to lark",
		// if uncommand Args line,will cause no suggesstion of "unknow command"
		//Args:  cobra.NoArgs,
		Version: "v0.0.1",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := viper.Unmarshal(&config.GlobalConfig); err != nil {
				log.Error().Msgf("unmarshal %s config failed: %v", cmd.Name(), err)
				return err
			}
			c, _ := json.Marshal(config.GlobalConfig)
			log.Info().Msg(string(c))
			if len(config.GlobalConfig.Kafka.Brokers) == 0 || config.GlobalConfig.Kafka.Topic == "" {
				return fmt.Errorf("invalid kafka config")
			}
			if config.GlobalConfig.Lark.AppID == "" || config.GlobalConfig.Lark.AppSecret == "" || config.GlobalConfig.Lark.ChatID == "" {
				return fmt.Errorf("invalid lark config")
			}
			return nil
		},
	}

	rootCmd.SetVersionTemplate("0.0.2")

	rootCmd.AddCommand(
		NewServerCmd(),
	)
	return rootCmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd := NewRootCmd()
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath("./config")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config.yaml")
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
