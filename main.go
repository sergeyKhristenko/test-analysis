package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

const (
	globSetting           = "test_globs"
	globEnv               = "PLUGIN_TEST_GLOBS"
	quarantineFileSetting = "quarantine_file"
	quarantineFileEnv     = "PLUGIN_QUARANTINE_FILE"
	quarantineSetting     = "fail_on_quarantine"
	quarantineEnv         = "PLUGIN_FAIL_ON_QUARANTINE"
)

func main() {
	app := &cli.App{
		Name:   "harness-parse-test-reports",
		Usage:  "Harness plugin to parse test reports",
		Action: run,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "test_globs",
				EnvVars: []string{"PLUGIN_TEST_GLOBS"},
			},
			&cli.StringFlag{
				Name:    "quarantine_file",
				EnvVars: []string{"PLUGIN_QUARANTINE_FILE"},
			},
			&cli.BoolFlag{
				Name:    "fail_on_quarantine",
				EnvVars: []string{"PLUGIN_FAIL_ON_QUARANTINE"},
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(c *cli.Context) error {
	p := Plugin{
		GlobPaths:        c.String(globSetting),
		QuarantineFile:   c.String(quarantineFileSetting),
		FailOnQuarantine: c.Bool(quarantineSetting),
	}
	return p.Exec()
}
