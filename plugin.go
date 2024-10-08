package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
)

type Plugin struct {
	GlobPaths        string
	QuarantineFile   string
	FailOnQuarantine bool
}

type TestStats struct {
	TestCount   int
	FailCount   int
	PassCount   int
	SkippedCount int
	ErrorCount  int
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

	var stats TestStats

	if p.FailOnQuarantine {
		if p.QuarantineFile == "" {
			log.Errorln(fmt.Errorf("fail_on_quarantine is true, but %s plugin setting or %s environment variable is not set", quarantineFileSetting, quarantineFileEnv))
			os.Exit(1)
		}

		quarantineList, err := LoadYAML(p.QuarantineFile)
		if err != nil {
			log.Errorln(fmt.Sprintf("Error loading quarantine file: %s", err))
			os.Exit(1)
		}

		stats, err = ParseTestsWithQuarantine(paths, quarantineList, log)
		if err != nil {
			log.Errorln(fmt.Sprintf("Error while parsing tests: %s", err))
			os.Exit(1)
		}
	} else {
		var err error
		stats, err = ParseTests(paths, log)
		if err != nil {
			log.Errorln(fmt.Sprintf("Error while parsing tests: %s", err))
			os.Exit(1)
		}
	}

	// Write output variables
	if err := WriteEnvToFile("TEST_COUNT", strconv.Itoa(stats.TestCount)); err != nil {
		log.Errorln(fmt.Sprintf("Error writing TEST_COUNT: %s", err))
	}
	if err := WriteEnvToFile("FAIL_COUNT", strconv.Itoa(stats.FailCount)); err != nil {
		log.Errorln(fmt.Sprintf("Error writing FAIL_COUNT: %s", err))
	}
	if err := WriteEnvToFile("PASS_COUNT", strconv.Itoa(stats.PassCount)); err != nil {
		log.Errorln(fmt.Sprintf("Error writing PASS_COUNT: %s", err))
	}
	if err := WriteEnvToFile("SKIPPED", strconv.Itoa(stats.SkippedCount)); err != nil {
		log.Errorln(fmt.Sprintf("Error writing SKIPPED: %s", err))
	}
	if err := WriteEnvToFile("ERROR_COUNT", strconv.Itoa(stats.ErrorCount)); err != nil {
		log.Errorln(fmt.Sprintf("Error writing ERROR_COUNT: %s", err))
	}

	log.Infoln(fmt.Sprintf("Final test statistics: Total: %d, Passed: %d, Failed: %d, Skipped: %d, Errors: %d",
		stats.TestCount, stats.PassCount, stats.FailCount, stats.SkippedCount, stats.ErrorCount))

	return nil
}

func WriteEnvToFile(key, value string) error {
	outputFile, err := os.OpenFile(os.Getenv("DRONE_OUTPUT"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open output file: %w", err)
	}
	defer outputFile.Close()
	_, err = fmt.Fprintf(outputFile, "%s=%s\n", key, value)
	if err != nil {
		return fmt.Errorf("failed to write to env: %w", err)
	}
	return nil
}