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
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile      string
	debug        bool
	dryRun       bool
	ignoreErrors bool
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "grafiti",
	Short: "A tool which ingests CloudTrail data and tags resources according to tagging rules.",
	Long:  `grafiti parses resource information out of CloudTrail events and applies tags from a config file to them. grafiti can then tag AWS resources with those tags, and can later delete those resources by those (or any other) tags.`,
	Run:   func(cmd *cobra.Command, args []string) { cmd.Usage() },
}

func init() {
	cobra.OnInitialize(initConfig)

	// Root config holds global config
	RootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "Config file (default is $HOME/.grafiti.toml)")
	RootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging.")
	RootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Output changes to stdout instead of AWS.")
	RootCmd.PersistentFlags().BoolVarP(&ignoreErrors, "ignore-errors", "e", false, "Continue processing even when there are API errors.")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	// Default bucket ejection time: 10 minutes in seconds
	viper.SetDefault("grafiti.bucketEjectLimitSeconds", 300)

	viper.SetConfigName(".grafiti") // name of config file (without extension)
	viper.AddConfigPath("$HOME")    // adding home directory as first search path
	viper.AutomaticEnv()            // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		logrus.Info("Using config file: ", viper.ConfigFileUsed())
	}

}

func main() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}
