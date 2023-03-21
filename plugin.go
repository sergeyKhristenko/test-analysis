package main

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

type Plugin struct {
	GlobPaths string
}

// Exec executes the plugin.
func (p Plugin) Exec() error {
	log := logrus.New()
	log.Out = os.Stdout

	if p.GlobPaths == "" {
		log.Errorln(fmt.Errorf("%s plugin setting or %s environment variable is not set", globSetting, globEnv))
		os.Exit(1)
	}

	paths := getPaths(p.GlobPaths)
	log.Infoln(fmt.Sprintf("Parsing test cases in globs: %s", paths))
	if err := ParseTests(paths, log); err != nil {
		log.Errorln(fmt.Sprintf("Error while parsing tests: %s", err))
		os.Exit(1)
	}
	return nil
}
