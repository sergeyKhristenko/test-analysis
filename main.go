package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

const (
	globSetting = "test_globs"
	globEnv     = "PLUGIN_TEST_GLOBS"
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
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(c *cli.Context) error {
	p := Plugin{
		GlobPaths: c.String(globSetting),
	}
	return p.Exec()
}
