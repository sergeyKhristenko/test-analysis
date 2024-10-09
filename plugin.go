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
	var err error

	if p.FailOnQuarantine {
		if p.QuarantineFile == "" {
			log.Errorln(fmt.Errorf("fail_on_quarantine is true, but %s plugin setting or %s environment variable is not set", quarantineFileSetting, quarantineFileEnv))
			os.Exit(1)
		}

		quarantineList, loadErr := LoadYAML(p.QuarantineFile)
		if loadErr != nil {
			log.Errorln(fmt.Sprintf("Error loading quarantine file: %s", loadErr))
			os.Exit(1)
		}

		stats, err = ParseTestsWithQuarantine(paths, quarantineList, log)
	} else {
		stats, err = ParseTests(paths, log)
	}

	// Always write output variables, even if there was an error
	writeTestStats(stats, log)

	log.Infoln(fmt.Sprintf("Final test statistics: Total: %d, Passed: %d, Failed: %d, Skipped: %d, Errors: %d",
		stats.TestCount, stats.PassCount, stats.FailCount, stats.SkippedCount, stats.ErrorCount))

	// Handle the error after writing stats
	if err != nil {
		log.Errorln(fmt.Sprintf("Error while parsing tests: %s", err))
		os.Exit(1)
	}

	return nil
}

func writeTestStats(stats TestStats, log *logrus.Logger) {
	statsMap := map[string]int{
		"TEST_COUNT":  stats.TestCount,
		"FAIL_COUNT":  stats.FailCount,
		"PASS_COUNT":  stats.PassCount,
		"SKIPPED":     stats.SkippedCount,
		"ERROR_COUNT": stats.ErrorCount,
	}

	for key, value := range statsMap {
		if err := WriteEnvToFile(key, strconv.Itoa(value), log); err != nil {
			log.Errorln(fmt.Sprintf("Error writing %s: %s", key, err))
		}
	}
}


func WriteEnvToFile(key, value string, log *logrus.Logger) error {
	outputFile, err := os.OpenFile(os.Getenv("DRONE_OUTPUT"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open output file: %w", err)
	}
	defer outputFile.Close()
	log.Infoln(fmt.Sprintf("Writing Test Stats %s : %s in func WriteEnvToFile to DRONE_OUTPUT",key,value))
	_, err = fmt.Fprintf(outputFile, "%s=%s\n", key, value)
	if err != nil {
		return fmt.Errorf("failed to write to env: %w", err)
	}
	return nil
}