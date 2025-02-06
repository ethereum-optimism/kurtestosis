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
)

type TestSuiteConfig struct {
	TestFilePattern string
	TestPattern     string
	TempDir         string
}

type TestFileRunner = func(testFile *TestFile) (StarlarkReporter, error)

type TestFunctionRunner = func(testFunction *TestFunction) (StarlarkReporter, error)

func RunTestSuite(project *KurtestosisProject, config TestSuiteConfig, reporter StarlarkReporter, testFileRunner TestFileRunner) error {
	// Let's now get the list of matching test files
	testFiles, testFilesErr := ListMatchingTestFiles(project, config.TestFilePattern)
	if testFilesErr != nil {
		return fmt.Errorf("error matching test files in project: %w", testFilesErr)
	}

	// Exit if there are no test suites to run
	if len(testFiles) == 0 {
		return nil
	}

	// Let the reporter know we're starting
	reporter.Before()

	// Run the test suites
	for _, testFile := range testFiles {
		testFileReporter, err := testFileRunner(testFile)
		if err != nil {
			return fmt.Errorf("error running test suite %s: %w", testFile, err)
		}

		reporter.Nest(testFileReporter)
	}

	// And let the reporter know we're done
	reporter.After()

	return nil
}

func CreateRunTestFile(config TestSuiteConfig, reporter StarlarkReporter, testFunctionRunner TestFunctionRunner) TestFileRunner {
	return func(testFile *TestFile) (StarlarkReporter, error) {
		// First we parse the test file and extract the names of matching test functions
		testFunctions, testFunctionsErr := ListMatchingTests(testFile, config.TestPattern)
		if testFunctionsErr != nil {
			return nil, fmt.Errorf("failed to list matching test functions in %s: %w", testFile, testFunctionsErr)
		}

		// Exit if there are no test suites to run
		if len(testFunctions) == 0 {
			return reporter, nil
		}

		// Let the reporter know we're starting
		reporter.Before()

		// Iterate over the test suites and run them one by one, collecting the test run summaries
		for _, testFunction := range testFunctions {
			testFunctionReporter, err := testFunctionRunner(testFunction)
			if err != nil {
				return nil, fmt.Errorf("failed to run test function %s: %w", testFunction, err)
			}

			// Add the test function reporter under the test file reporter
			reporter.Nest(testFunctionReporter)
		}

		// And let the reporter know we're done
		reporter.After()

		return reporter, nil
	}
}

func CreateRunTestFunction(config TestSuiteConfig, reporter StarlarkReporter) TestFunctionRunner {
	return func(testFunction *TestFunction) (StarlarkReporter, error) {
		var err error

		// Let's make a database first
		enclaveDB, teardownEnclaveDB, err := backend.CreateEnclaveDB()
		if err != nil {
			return nil, fmt.Errorf("failed to create EnclaveDB: %w", err)
		}

		// We want to tear the database down once it's all over
		defer teardownEnclaveDB()

		// Package content providers
		localGitPackageContentProvider, err := backend.CreateLocalGitPackageContentProvider(config.TempDir, enclaveDB)
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

		reporter.Before()

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

		reporter.After()

		return reporter, nil
	}
}
