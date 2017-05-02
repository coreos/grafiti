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

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var debug bool
var dryRun bool
var ignoreErrors bool

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "grafiti",
	Short: "A tool which ingests CloudTrail data and tags resources according to tagging rules.",
	Long: `grafiti tags CloudTrail tags as input and tags AWS resources in response. By default, this tags resources
with the creating user.`,
	Run: func(cmd *cobra.Command, args []string) { cmd.Usage() },
}

func init() {
	cobra.OnInitialize(initConfig)

	// Root config holds global config
	RootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.grafiti.toml)")
	RootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable debug logging")
	RootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "output changes to stdout instead of AWS")
	RootCmd.PersistentFlags().BoolVarP(&ignoreErrors, "ignoreErrors", "e", false, "Continue processing even when there are API errors.")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName(".grafiti") // name of config file (without extension)
	viper.AddConfigPath("$HOME")    // adding home directory as first search path
	viper.AutomaticEnv()            // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Info("Using config file:", viper.ConfigFileUsed())
	}

}

func main() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}
