// Copyright © 2017 grafiti/predator authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile      string
	debug        bool
	dryRun       bool
	ignoreErrors bool
)

// Grafiti-specific environment variables are prefixed with GRF_
var envVarMap = map[string]string{
	"AWS_REGION":          "region",
	"GRF_START_HOUR":      "startHour",
	"GRF_END_HOUR":        "endHour",
	"GRF_START_TIMESTAMP": "startTimeStamp",
	"GRF_END_TIMESTAMP":   "endTimeStamp",
	"GRF_INCLUDE_EVENT":   "includeEvent",
	"GRF_MAX_NUM_RETRIES": "maxNumRequestRetries",
}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "grafiti",
	Short: "Ingest CloudTrail data to tag, then delete, AWS resources.",
	Long:  `Parse resources ID's from CloudTrail events to tag them in AWS, then delete them later.`,
	Run:   func(cmd *cobra.Command, args []string) { cmd.Usage() },
}

func init() {
	cobra.OnInitialize(initConfig)

	// Root config holds global config
	RootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "Config file (default: $HOME/.grafiti.toml)")
	RootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging.")
	RootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Output changes to stdout instead of AWS.")
	RootCmd.PersistentFlags().BoolVarP(&ignoreErrors, "ignore-errors", "e", false, "Continue processing even when there are API errors.")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Check for 'jq' in path
	if path, err := exec.LookPath("jq"); err != nil || path == "" {
		fmt.Println("Please install 'jq' before running grafiti.")
		os.Exit(1)
	}

	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath("$HOME")    // adding home directory as first search path
		viper.SetConfigName(".grafiti") // name of config file (without extension)
	}

	// Default bucket ejection time: 10 minutes in seconds
	viper.SetDefault("bucketEjectLimitSeconds", 300)
	// Default number of delete request retries
	viper.SetDefault("maxNumRequestRetries", 8)

	// Prefer env variables over config file fields
	for ev, path := range envVarMap {
		if val := os.Getenv(ev); val != "" {
			viper.Set(path, val)
		}
	}

	// If a config file is found, read in its data
	if err := viper.ReadInConfig(); err != nil {
		// Only log if a config file was provided but not found. The user probably
		// wants to solely use env vars to configure grafiti if no config file was
		// provided
		if cfgFile != "" {
			logrus.Errorln("read config file:", err)
			os.Exit(1)
		}
	} else {
		logrus.Infoln("Using config file:", viper.ConfigFileUsed())
		return
	}

	// No config file found so configure grafiti with a dummy config file and
	// environment variables
	viper.SetConfigType("toml")
	if err := viper.ReadConfig(bytes.NewBuffer([]byte(""))); err != nil {
		logrus.Errorln("read dummy config file:", err)
		os.Exit(1)
	} else {
		logrus.Info("Using environment variables to configure grafiti.")
		return
	}
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}
