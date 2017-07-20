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
	"errors"
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

// grafiti-specific environment variables will begin with GRF_
var envVarMap = map[string]string{
	"AWS_REGION":          "region",
	"GRF_START_HOUR":      "startHour",
	"GRF_END_HOUR":        "endHour",
	"GRF_START_TIMESTAMP": "startTimeStamp",
	"GRF_END_TIMESTAMP":   "endTimeStamp",
	"GRF_INCLUDE_EVENT":   "includeEvent",
}

// http://tldp.org/LDP/abs/html/exitcodes.html
const (
	exitSuccess = iota
	exitError
	exitInvalidInput
	exitBadArgs = 128
)

func exitWithError(code int, err error) {
	fmt.Fprintln(os.Stderr, "Error:", err)
	os.Exit(code)
}

func exitWithSuccess() {
	os.Exit(exitSuccess)
}

// logger prints errors in JSON format without a timestamp
var logger = logrus.Logger{
	Out:       os.Stderr,
	Formatter: &logrus.JSONFormatter{DisableTimestamp: true},
	Level:     logrus.InfoLevel,
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
		exitWithError(exitError, errors.New("'jq' must be installed before running grafiti"))
	}

	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath("$HOME")    // adding home directory as first search path
		viper.SetConfigName(".grafiti") // name of config file (without extension)
	}

	// Use env variables as defaults
	for ev, path := range envVarMap {
		val := os.Getenv(ev)
		if val != "" {
			viper.SetDefault(path, val)
		}
	}

	// Default bucket ejection time: 10 minutes in seconds
	viper.SetDefault("bucketEjectLimitSeconds", 300)
	// Default number of delete request retries
	viper.SetDefault("maxNumRequestRetries", 8)

	// If a config file is found, read in its data
	if err := viper.ReadInConfig(); err == nil {
		logrus.Info("Using config file: ", viper.ConfigFileUsed())
		return
	}

	// No config file found so configure grafiti with a dummy config file (Reader)
	// and environment variables
	viper.SetConfigType("toml")
	if err := viper.ReadConfig(bytes.NewBuffer([]byte(""))); err == nil {
		logrus.Info("Using environment variables to configure grafiti.")
		return
	}

}

func main() {
	if err := RootCmd.Execute(); err != nil {
		exitWithError(exitError, err)
	}
}
