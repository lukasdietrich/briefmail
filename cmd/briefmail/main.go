// Copyright (C) 2019  Lukas Dietrich <lukas@lukasdietrich.com>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const usageText = `
Usage:
  briefmail [OPTIONS] COMMAND

  Briefly set up email.

Version:
  %s

Commands:
  start     Start the briefmail server
  shell     Start an interactive administration shell

Options:
`

var (
	// Version is set at compile-time.
	Version string
)

func init() {
	viper.SetDefault("log.level", "debug")

	logrus.SetReportCaller(true)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:    true,
		QuoteEmptyFields: true,
		CallerPrettyfier: func(frame *runtime.Frame) (string, string) {
			return frame.Func.Name(), ""
		},
	})
}

func main() {
	var (
		configFilename string
	)

	flag.StringVar(&configFilename,
		"config",
		"config.toml",
		"Path to configuration file")
	flag.Usage = printUsage
	flag.Parse()

	switch commandName := flag.Arg(0); commandName {
	case "start", "shell":
		setupConfig(configFilename)
		setupLogger()
		runCommand(commandName)
	default:
		flag.Usage()
	}
}

type command interface {
	run() error
}

func runCommand(commandName string) {
	var (
		cmd command
		err error
	)

	switch commandName {
	case "start":
		cmd, err = newStartCommand()
	case "shell":
		cmd, err = newShellCommand()
	}

	if err != nil {
		logrus.Fatalf("could not initialize the application: %v", err)
	}

	if err := cmd.run(); err != nil {
		logrus.Fatalf("%v", err)
	}
}

func printUsage() {
	fmt.Fprintf(flag.CommandLine.Output(), usageText, Version)
	flag.PrintDefaults()
}

func setupLogger() {
	logLevel, err := logrus.ParseLevel(viper.GetString("log.level"))
	if err != nil {
		logrus.Fatalf("unknown log level: %v", err)
	}

	logrus.SetLevel(logLevel)
}

func setupConfig(filename string) {
	logrus.Infof("loading config file %s", filename)

	viper.SetTypeByDefaultValue(true)
	viper.SetEnvKeyReplacer(strings.NewReplacer("_", "."))
	viper.SetEnvPrefix("BRIEFMAIL")
	viper.AutomaticEnv()
	viper.SetConfigFile(filename)

	if err := viper.ReadInConfig(); err != nil {
		if os.IsNotExist(err) {
			logrus.Warnf("configuration file missing: %v", err)
		} else {
			logrus.Fatalf("could not load configuration: %v", err)
		}
	}
}
