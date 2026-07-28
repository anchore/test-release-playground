package main

import (
	"os"

	"github.com/anchore/release-playground/internal/cli"
)

const valueNotProvided = "[not provided]"

// all variables here are provided as build-time arguments, with clear default values
var (
	version        = valueNotProvided
	gitCommit      = valueNotProvided
	gitDescription = valueNotProvided
	buildDate      = valueNotProvided
)

func main() {
	app := cli.New(cli.Identification{
		Version:        version,
		GitCommit:      gitCommit,
		GitDescription: gitDescription,
		BuildDate:      buildDate,
	})

	if err := app.Run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}
