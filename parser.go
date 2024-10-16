package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/harness-community/parse-test-reports/gojunit"
	"github.com/mattn/go-zglob"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

func getPaths(globVal string) []string {
	paths := make([]string, 0)
	globValSplit := strings.Split(globVal, ",")
	for _, val := range globValSplit {
		if val == "" {
			continue
		}
		val = strings.TrimSpace(val)
		paths = append(paths, val)
	}
	return paths
}

// ParseTests parses XMLs and returns error if there are any failures
func ParseTests(paths []string, log *logrus.Logger) (TestStats, error) {
	files := getFiles(paths, log)
	stats := TestStats{}

	if len(files) == 0 {
		log.Errorln("could not find any files matching the provided report path")
		return stats, nil
	}

	for _, file := range files {
		suites, err := gojunit.IngestFile(file)
		if err != nil {
			log.WithError(err).WithField("file", file).
				Errorln(fmt.Sprintf("could not parse file %s", file))
			continue
		}
		fileStats := TestStats{}
		for _, suite := range suites {
			for _, test := range suite.Tests {
				fileStats.TestCount++
				switch test.Result.Status {
				case "passed":
					fileStats.PassCount++
				case "failed":
					fileStats.FailCount++
				case "skipped":
					fileStats.SkippedCount++
				case "error":
					fileStats.ErrorCount++
				}
			}
		}
		log.Infoln(fmt.Sprintf("File %s processed. Stats: Total: %d, Passed: %d, Failed: %d, Skipped: %d, Errors: %d",
			file, fileStats.TestCount, fileStats.PassCount, fileStats.FailCount, fileStats.SkippedCount, fileStats.ErrorCount))
		
		// Aggregate stats
		stats.TestCount += fileStats.TestCount
		stats.PassCount += fileStats.PassCount
		stats.FailCount += fileStats.FailCount
		stats.SkippedCount += fileStats.SkippedCount
		stats.ErrorCount += fileStats.ErrorCount
	}

	if stats.FailCount > 0 || stats.ErrorCount > 0 {
		return stats, fmt.Errorf("found %d failed tests and %d errors", stats.FailCount, stats.ErrorCount)
	}
	return stats, nil
}

// getFiles returns uniques file paths provided in the input after expanding the input paths
func getFiles(paths []string, log *logrus.Logger) []string {
	var files []string
	for _, p := range paths {
		path, err := expandTilde(p)
		if err != nil {
			log.WithError(err).WithField("path", p).
				Errorln("errored while trying to expand paths")
			continue
		}
		matches, err := zglob.Glob(path)
		if err != nil {
			log.WithError(err).WithField("path", path).
				Errorln("errored while trying to resolve path regex")
			continue
		}

		files = append(files, matches...)
	}
	return uniqueItems(files)
}

func uniqueItems(items []string) []string {
	var result []string

	set := make(map[string]bool)
	for _, item := range items {
		if _, ok := set[item]; !ok {
			result = append(result, item)
			set[item] = true
		}
	}
	return result
}

// expandTilde method expands the given file path to include the home directory
// if the path is prefixed with `~`. If it isn't prefixed with `~`, the path is
// returned as-is.
func expandTilde(path string) (string, error) {
	if path == "" {
		return path, nil
	}

	if path[0] != '~' {
		return path, nil
	}

	if len(path) > 1 && path[1] != '/' && path[1] != '\\' {
		return "", errors.New("cannot expand user-specific home dir")
	}

	dir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to fetch home directory: %s", err)
	}
	return filepath.Join(dir, path[1:]), nil
}

// LoadYAML reads a YAML file from either a public URL or a local file and returns the data as a map
func LoadYAML(source string) (map[string]interface{}, error) {
	log := logrus.New()
	log.Infoln(fmt.Sprintf("Loading YAML from source: %s", source))

	var data []byte
	var err error

	// Check if source is a URL
	if isURL(source) {
		// Fetch the YAML content from the URL
		resp, err := http.Get(source)
		if err != nil {
			log.WithError(err).Errorln("Failed to fetch YAML from URL")
			return nil, fmt.Errorf("failed to fetch YAML from URL: %w", err)
		}
		defer resp.Body.Close()

		data, err = io.ReadAll(resp.Body)
		if err != nil {
			log.WithError(err).Errorln("Failed to read YAML data from URL")
			return nil, fmt.Errorf("failed to read YAML data from URL: %w", err)
		}
	} else {
		// Read the YAML content from the local file
		data, err = os.ReadFile(source)
		if err != nil {
			log.WithError(err).Errorln("Failed to read local YAML file")
			return nil, fmt.Errorf("failed to read YAML file: %w", err)
		}
	}

	// Parse YAML data
	var result map[string]interface{}
	err = yaml.Unmarshal(data, &result)
	if err != nil {
		log.WithError(err).Errorln("Failed to parse YAML")
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	log.Infoln("Successfully loaded and parsed YAML")
	return result, nil
}

// Helper function to check if the source is a URL
func isURL(source string) bool {
	return len(source) > 4 && source[:4] == "http"
}

// ParseTestsWithQuarantine parses XMLs and fails only if errors are found in non-quarantined tests
func ParseTestsWithQuarantine(paths []string, quarantineList map[string]interface{}, log *logrus.Logger) (TestStats, error) {
	files := getFiles(paths, log)
	stats := TestStats{}
	nonQuarantinedFailures := 0
	expiredTests := 0

	if len(files) == 0 {
		log.Errorln("could not find any files matching the provided report path")
		return stats, nil
	}

	log.Infoln("Starting to parse tests with quarantine list")

	for _, file := range files {
		suites, err := gojunit.IngestFile(file)
		if err != nil {
			log.WithError(err).WithField("file", file).
				Errorln(fmt.Sprintf("could not parse file %s", file))
			continue
		}
		fileStats := TestStats{}
		for _, suite := range suites {
			for _, test := range suite.Tests {
				fileStats.TestCount++
				testIdentifier := fmt.Sprintf("%s.%s", test.Classname, test.Name)
				switch test.Result.Status {
				case "passed":
					fileStats.PassCount++
				case "failed":
					// Check if the test is quarantined
					if !isQuarantined(testIdentifier, quarantineList, log) {
						log.Infoln(fmt.Sprintf("Not Quarantined test failed: %s", testIdentifier))
						nonQuarantinedFailures++
					} else {
						log.Infoln(fmt.Sprintf("Quarantined test failed: %s", testIdentifier))

						// Check if the test is expired
						if isExpired(testIdentifier, quarantineList, log) {
							log.Infoln(fmt.Sprintf("Quarantined test expired: %s", testIdentifier))
							expiredTests++
						}
					}
					fileStats.FailCount++
				case "skipped":
					fileStats.SkippedCount++
				case "error":
					fileStats.ErrorCount++
					// nonQuarantinedFailures++ // Assuming errors are always considered non-quarantined
				}
			}
		}
		log.Infoln(fmt.Sprintf("File %s processed. Stats: Total: %d, Passed: %d, Failed: %d, Skipped: %d, Errors: %d",
			file, fileStats.TestCount, fileStats.PassCount, fileStats.FailCount, fileStats.SkippedCount, fileStats.ErrorCount))
		
		// Aggregate stats
		stats.TestCount += fileStats.TestCount
		stats.PassCount += fileStats.PassCount
		stats.FailCount += fileStats.FailCount
		stats.SkippedCount += fileStats.SkippedCount
		stats.ErrorCount += fileStats.ErrorCount
	}

	log.Infoln("Finished parsing tests with quarantine list")

	if nonQuarantinedFailures > 0 || expiredTests > 0 {
		return stats, fmt.Errorf("found %d non-quarantined failed tests and %d expired tests", nonQuarantinedFailures, expiredTests)
	}
	return stats, nil
}

// isQuarantined checks if the test is present in the quarantine list and respects the date range
func isQuarantined(testIdentifier string, quarantineList map[string]interface{}, log *logrus.Logger) bool {
	log.Infoln(fmt.Sprintf("Checking if test is quarantined: %s", testIdentifier))

	tests, ok := quarantineList["quarantine_tests"].([]interface{})
	if !ok {
		log.Warnln("Quarantine list does not contain 'quarantine_tests' key or it's not a slice")
		return false
	}

	for _, test := range tests {
		if testMap, ok := test.(map[interface{}]interface{}); ok {
			quarantinedClassname, classnameOk := testMap["classname"].(string)
			quarantinedName, nameOk := testMap["name"].(string)

			if classnameOk && nameOk {
				quarantinedIdentifier := quarantinedClassname + "." + quarantinedName
				if quarantinedIdentifier == testIdentifier {
					log.Infoln(fmt.Sprintf("Test %s is quarantined", testIdentifier))
					return true
				}
			}
		}
	}

	log.Infoln(fmt.Sprintf("Test %s is not quarantined", testIdentifier))
	return false
}

// isExpired checks if the current date is outside the start_date and end_date for a quarantined test
func isExpired(testIdentifier string, quarantineList map[string]interface{}, log *logrus.Logger) bool {
	tests, ok := quarantineList["quarantine_tests"].([]interface{})
	if !ok {
		log.Warnln("Quarantine list does not contain 'quarantine_tests' key or it's not a slice")
		return false
	}

	for _, test := range tests {
		if testMap, ok := test.(map[interface{}]interface{}); ok {
			quarantinedClassname, classnameOk := testMap["classname"].(string)
			quarantinedName, nameOk := testMap["name"].(string)

			if classnameOk && nameOk {
				quarantinedIdentifier := quarantinedClassname + "." + quarantinedName
				if quarantinedIdentifier == testIdentifier {
					// Check for date range
					startDate, startOk := testMap["start_date"].(string)
					endDate, endOk := testMap["end_date"].(string)

					if startOk && endOk {
						currentDate := time.Now()

						startTime, err := time.Parse("2006-01-02", startDate)
						if err != nil {
							log.WithError(err).Warnln("Failed to parse start_date")
							continue
						}

						endTime, err := time.Parse("2006-01-02", endDate)
						if err != nil {
							log.WithError(err).Warnln("Failed to parse end_date")
							continue
						}

						if currentDate.Before(startTime) || currentDate.After(endTime) {
							log.Infoln(fmt.Sprintf("Current Date %s lies outside start_date %s and end_date %s.", currentDate, startTime, endTime))
							return true
						}
					}
				}
			}
		}
	}

	return false
}