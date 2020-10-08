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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/lukasdietrich/briefmail/internal/log"
)

const usageText = `
Usage:
  briefmail [OPTIONS] COMMAND

  Briefly set up email.

Version:
  %s

Commands:
  start     Start the briefmail server.
  shell     Start an interactive administration shell.

Options:
%s
`

var (
	// Version is set at compile-time.
	Version string
)

func init() {
	viper.SetDefault("log.level", "debug")
	viper.SetDefault("general.hostname", "localhost")
}

func main() {
	var (
		configFilename      string
		enableConsoleLogger bool
	)

	flags := pflag.NewFlagSet("briefmail", pflag.ContinueOnError)
	flags.Usage = printUsage(flags)

	flags.StringVarP(&configFilename,
		"config", "c",
		"",
		"Path to a configuration file.")

	flags.BoolVar(&enableConsoleLogger,
		"pretty-console-logger",
		false,
		"Enable pretty, but inefficient console logger.")

	if err := flags.Parse(os.Args); err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			return
		}

		fmt.Println(err)
		os.Exit(1)
	}

	switch commandName := flags.Arg(1); commandName {
	case "shell":
		enableConsoleLogger = true
		fallthrough
	case "start":
		if enableConsoleLogger {
			log.Logger = log.Logger.Output(zerolog.NewConsoleWriter())
		}

		setupConfig(configFilename)
		setupLogLevel()
		runCommand(commandName)
	default:
		flags.Usage()
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
		printConfig()
		cmd, err = newStartCommand()
	case "shell":
		cmd, err = newShellCommand()
	}

	if err != nil {
		log.Fatal().
			Err(err).
			Msg("could not initialize the application")
	}

	if err := cmd.run(); err != nil {
		log.Fatal().
			Err(err).
			Msg("error during execution")
	}
}

func printUsage(flags *pflag.FlagSet) func() {
	return func() {
		fmt.Fprintf(os.Stderr, usageText,
			Version,
			flags.FlagUsages())
	}
}

func setupConfig(filename string) {
	viper.SetTypeByDefaultValue(true)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix("BRIEFMAIL")

	if filename != "" {
		readConfig(filename)
	} else {
		log.Info().Msg("no config file provided. using environment only")
	}
}

func readConfig(filename string) {
	log.Info().
		Str("filename", filename).
		Msg("loading configuration from file")

	viper.SetConfigFile(filename)

	if err := viper.ReadInConfig(); err != nil {
		if os.IsNotExist(err) {
			log.Fatal().
				Err(err).
				Msg("configuration file missing")
		} else {
			log.Fatal().
				Err(err).
				Msg("could not load configuration")
		}
	}
}

func printConfig() {
	keys := viper.AllKeys()
	sort.Strings(keys)

	for _, key := range keys {
		v, _ := json.Marshal(viper.Get(key))
		log.Debug().
			Str("key", key).
			RawJSON("value", v).
			Msg("configuration")
	}
}

func setupLogLevel() {
	logLevelName := viper.GetString("log.level")
	logLevelName = strings.ToLower(logLevelName)

	logLevel, err := zerolog.ParseLevel(logLevelName)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("unknown log level")
	}

	zerolog.SetGlobalLevel(logLevel)
}
