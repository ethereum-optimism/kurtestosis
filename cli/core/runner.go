package core

import (
	"context"
	"fmt"
	"kurtestosis/cli/kurtosis"
	"kurtestosis/cli/kurtosis/backend"

	"github.com/kurtosis-tech/kurtosis/container-engine-lib/lib/backend_interface/objects/image_download_mode"
	"github.com/kurtosis-tech/kurtosis/core/server/api_container/server/startosis_engine/enclave_structure"
	"github.com/kurtosis-tech/kurtosis/core/server/api_container/server/startosis_engine/instructions_plan/resolver"
	"github.com/kurtosis-tech/kurtosis/core/server/api_container/server/startosis_engine/startosis_constants"
	"github.com/sirupsen/logrus"
)

type TestSuiteConfig struct {
	TestFilePattern string
	TestPattern     string
}

func RunTestSuite(config TestSuiteConfig, project *KurtestosisProject, logger *logrus.Logger) (StarlarkReporter, error) {
	// Let's now get the list of matching test files
	testFiles, testFilesErr := ListMatchingTestFiles(project, config.TestFilePattern)
	if testFilesErr != nil {
		logger.Errorf("Error matching test files in project: %v", testFilesErr)

		return nil, fmt.Errorf("error matching test files in project: %w", testFilesErr)
	}

	// Create a test reporter to hold the test status
	reporter := CreateStarlarkTestSuiteReporter(project, logger)

	// Exit if there are no test suites to run
	if len(testFiles) == 0 {
		logger.Warn("No test suites found matching the glob pattern")

		return reporter, nil
	}

	// Let the reporter know we're starting
	reporter.Before()

	// Run the test suites
	for _, testFile := range testFiles {
		testFileReporter, err := RunTestFile(testFile, testFileReporter)
		if err != nil {
			return nil, fmt.Errorf("error running test suite %s: %w", testFile, err)
		}

		reporter.Nest(testFileReporter)
	}

	// And let the reporter know we're done
	reporter.After()

	return reporter, nil
}

func RunTestFile(config TestSuiteConfig, testFile *TestFile, logger *logrus.Logger) (StarlarkReporter, error) {
	// First we parse the test file and extract the names of matching test functions
	testFunctions, testFunctionsErr := ListMatchingTests(testFile, testPatternStr)
	if testFunctionsErr != nil {
		return nil, fmt.Errorf("failed to list matching test functions in %s: %w", testFile, testFunctionsErr)
	}

	// Create a test reporter to hold the test status
	reporter := CreateStarlarkTestFileReporter(testFile, logger)

	// Exit if there are no test suites to run
	if len(testFunctions) == 0 {
		logrus.Warnf("No tests found matching the test pattern %s in %s", testPatternStr, testFile)

		return reporter, nil
	}

	// Let the reporter know we're starting
	reporter.Before()

	// Iterate over the test suites and run them one by one, collecting the test run summaries
	for _, testFunction := range testFunctions {
		testFunctionReporter, err := RunTestFunction(testFunction, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to run test function %s: %w", testFunction, testFunctionErr)
		}

		// Add the test function reporter under the test file reporter
		reporter.Nest(testFunctionReporter)
	}

	// And let the reporter know we're done
	reporter.After()

	return reporter, nil
}

func RunTestFunction(testFunction *TestFunction, logger *logrus.Logger) (StarlarkReporter, error) {
	var err error

	// Create a test reporter to hold the test status
	reporter := CreateStarlarkTestFunctionReporter(testFunction, logger)

	// Let's make a database first
	enclaveDB, teardownEnclaveDB, err := backend.CreateEnclaveDB()
	if err != nil {
		return nil, fmt.Errorf("failed to create EnclaveDB: %w", err)
	}

	// We want to tear the database down once it's all over
	defer teardownEnclaveDB()

	// Package content providers
	localGitPackageContentProvider, err := backend.CreateLocalGitPackageContentProvider(tempDirRootStr, enclaveDB)
	if err != nil {
		return nil, fmt.Errorf("failed to create local git package content provider: %w", err)
	}
	localProxyPackageContentProvider := backend.CreateLocalProxyPackageContentProvider(testFunction.TestFile.Project, localGitPackageContentProvider)

	// Now we create the value storage that holds all the starlark values
	starlarkValueSerde := backend.CreateStarlarkValueSerde()
	runtimeValueStore, interpretationTimeValueStore, err := backend.CreateValueStores(enclaveDB, starlarkValueSerde)
	if err != nil {
		return nil, fmt.Errorf("failed to create kurtosis value stores: %w", err)
	}

	// We load all the kurtestosis-specific predeclared starlark builtins
	predeclared, err := kurtosis.LoadKurtestosisPredeclared()
	if err != nil {
		return nil, err
	}

	// And we create a processor function that merges them with kurtosis predeclared builtins
	processBuiltins := kurtosis.CreateProcessBuiltins(predeclared)

	// We setup a test reporter
	//
	// Besides collecting and formatting the test output (mostly TBD),
	// a reporter is required for correct functioning of the starlarktest assert module
	kurtosis.SetupKurtestosisPredeclared(reporter)

	// Service network (99% mock)
	serviceNetwork := backend.CreateKurtestosisServiceNetwork()

	// And finally an interpreter
	interpreter, err := backend.CreateInterpreter(
		localProxyPackageContentProvider, // packageContentProvider
		starlarkValueSerde,               // starlarkValueSerde
		runtimeValueStore,                // runtimeValueStore
		interpretationTimeValueStore,     // interpretationTimeValueStore
		processBuiltins,                  // processBuiltins
		serviceNetwork,                   // serviceNetwork
	)
	if err != nil {
		return nil, err
	}

	testSuiteScript, mainFunctionName, inputArgs := kurtosis.WrapTestFunction(testFunction)

	_, _, interpretationErr := interpreter.Interpret(
		context.Background(), // context
		testFunction.TestFile.Project.KurotosisYml.PackageName, // packageId
		mainFunctionName, // mainFunctionName
		testFunction.TestFile.Project.KurotosisYml.PackageReplaceOptions, // packageReplaceOptions
		startosis_constants.PlaceHolderMainFileForPlaceStandAloneScript,  // relativePathtoMainFile
		testSuiteScript,                          // serializedStarlark
		inputArgs,                                // serializedJsonParams
		false,                                    // nonBlockingMode
		enclave_structure.NewEnclaveComponents(), // enclaveComponents
		resolver.NewInstructionsPlanMask(0),      // instructionsPlanMask
		image_download_mode.ImageDownloadMode_Missing, // imageDownloadMode
	)

	// Any interpretation errors will be sent to the reporter as well
	if interpretationErr != nil {
		reporter.Error(interpretationErr)
	}

	return reporter, nil
}
