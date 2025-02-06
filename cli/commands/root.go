package commands

import (
	"fmt"
	"strings"

	"kurtestosis/cli/core"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	// CLI Flag names
	logLevelStrFlag        = "log-level"
	tempDirRootStrFlag     = "temp-dir"
	testFilePatternStrFlag = "test-file-pattern"
	testPatternStrFlag     = "test-pattern"
)

// The variables configurable using CLI flags
var (
	// Log level for the CLI
	logLevelStr string

	// Temporary directory in which to store kurtosis' temporary filesystem
	tempDirRootStr string

	// Glob pattern to use when looking for test files
	testFilePatternStr string

	// Glob pattern to use when looking for test functions
	testPatternStr string
)

// RootCmd Suppressing exhaustruct requirement because this struct has ~40 properties
// nolint: exhaustruct
var RootCmd = &cobra.Command{
	Use:   KurtestosisCmdStr,
	Short: "Kurtestosis, Kurtosis test runner CLI",
	// Cobra will print usage whenever _any_ error occurs, including ones we throw in Kurtosis
	// This doesn't make sense in 99% of the cases, so just turn them off entirely
	SilenceUsage: true,
	// Cobra prints the errors itself, however, with this flag disabled it give Kurtosis control
	// and allows us to post process the error in the main.go file.
	SilenceErrors: true,
	// The PersistentPreRunE hook runs before every descendant command
	// and will setup things like log level
	PersistentPreRunE: setupCLI,
	RunE:              run,
	Args:              cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
}

func init() {
	RootCmd.PersistentFlags().StringVar(
		&logLevelStr,
		logLevelStrFlag,
		logrus.InfoLevel.String(),
		"Sets the level that the CLI will log at ("+strings.Join(getAllLogLevelStrings(), "|")+")",
	)

	RootCmd.Flags().StringVar(
		&tempDirRootStr,
		tempDirRootStrFlag,
		KurtestosisDefaultTempDirRoot,
		"Directory for kurtosis temporary files",
	)

	RootCmd.Flags().StringVar(
		&testFilePatternStr,
		testFilePatternStrFlag,
		KurtestosisDefaultTestFilePattern,
		"Glob expression to use when looking for starlark test files",
	)

	RootCmd.Flags().StringVar(
		&testPatternStr,
		testPatternStrFlag,
		KurtestosisDefaultTestFunctionPattern,
		"Glob expression to use when looking for test functions",
	)
}

func run(cmd *cobra.Command, args []string) error {
	logger := logrus.StandardLogger()

	logger.Warn("kurtestosis CLI is still work in progress")

	// First we load the project
	projectPath := args[0]
	project, projectErr := core.LoadKurtestosisProject(args[0])
	if projectErr != nil {
		logger.Errorf("Failed to load project from %s: %v", projectPath, projectErr)

		return fmt.Errorf("failed to load project from %s: %w", projectPath, projectErr)
	}

	testSuiteConfig := &core.TestSuiteConfig{
		TestFilePattern: testFilePatternStr
		TestPattern: testPatternStr,
	}

	reporter, err := core.RunTestSuite(project, testSuiteConfig, logger)

	// Let's now get the list of matching test files
	testFiles, testFilesErr := core.ListMatchingTestFiles(project, testFilePatternStr)
	if testFilesErr != nil {
		logger.Errorf("Error matching test files in project: %v", testFilesErr)

		return fmt.Errorf("error matching test files in project: %w", testFilesErr)
	}

	// Exit if there are no test suites to run
	if len(testFiles) == 0 {
		logger.Warn("No test suites found matching the glob pattern")

		return nil
	}

	// Create a reporter for the test run
	testSuiteReporter := core.CreateStarlarkTestSuiteReporter(project, logger)

	// Call the
	testSuiteReporter.Before()

	// Run the test suites
	for _, testFile := range testFiles {
		testFileReporter := testSuiteReporter.Nest(core.CreateStarlarkTestFileReporter(testFile, logger))

		err := runTestFile(testFile, testFileReporter)
		if err != nil {
			logrus.Errorf("error running test suite %s: %v", testFile, err)

			return fmt.Errorf("error running test suite %s: %w", testFile, err)
		}
	}

	if testSuiteReporter.Success() {
		return nil
	}

	return fmt.Errorf("test suite failed")
}

// Concatenates all logrus log level strings into a string array
func getAllLogLevelStrings() []string {
	result := []string{}
	for _, level := range logrus.AllLevels {
		levelStr := level.String()
		result = append(result, levelStr)
	}
	return result
}

// Setup function to run before any command execution
func setupCLI(cmd *cobra.Command, args []string) error {
	// First we configure the log level
	logLevel, err := logrus.ParseLevel(logLevelStr)
	if err != nil {
		return fmt.Errorf("error parsing the %s CLI argument: %w", logLevelStrFlag, err)
	}

	logrus.SetOutput(cmd.OutOrStdout())
	logrus.SetLevel(logLevel)

	return nil
}
