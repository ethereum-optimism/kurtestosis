package kurtosis

import (
	"fmt"
	"kurtestosis/cli/core"
	"kurtestosis/cli/kurtosis/backend"

	"github.com/kurtosis-tech/kurtosis/core/server/api_container/server/startosis_engine"
	"github.com/kurtosis-tech/kurtosis/core/server/api_container/server/startosis_engine/startosis_constants"
	"go.starlark.net/starlarktest"
)

// Creates a wrapper script that executes testFunction
// using the kurtestosis starlark module
//
// This module sets up necessary starlark runtime (especially for the assert module)
func WrapTestFunction(testFunction *core.TestFunction) (starlark string, mainFunctionName string, jsonInputArgs string) {
	return fmt.Sprintf(`
sut = import_module("/%s")

def run(plan):
	kurtestosis.test(plan, sut, "%s")
`, testFunction.TestFile.Path, testFunction.Name), "run", startosis_constants.EmptyInputArgs
}

type TeardownInterpreter = func()

type SetupInterpreter = func(reporter starlarktest.Reporter) (*startosis_engine.StartosisInterpreter, TeardownInterpreter, error)

func CreateSetupInterpreter(project *core.KurtestosisProject, config *core.TestSuiteConfig) SetupInterpreter {
	return func(reporter starlarktest.Reporter) (*startosis_engine.StartosisInterpreter, TeardownInterpreter, error) {
		var err error

		// Let's make a database first
		enclaveDB, teardownEnclaveDB, err := backend.CreateEnclaveDB()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create EnclaveDB: %w", err)
		}

		// Package content providers
		localGitPackageContentProvider, err := backend.CreateLocalGitPackageContentProvider(config.TempDir, enclaveDB)
		if err != nil {
			return nil, teardownEnclaveDB, fmt.Errorf("failed to create local git package content provider: %w", err)
		}
		localProxyPackageContentProvider := backend.CreateLocalProxyPackageContentProvider(project, localGitPackageContentProvider)

		// Now we create the value storage that holds all the starlark values
		starlarkValueSerde := backend.CreateStarlarkValueSerde()
		runtimeValueStore, interpretationTimeValueStore, err := backend.CreateValueStores(enclaveDB, starlarkValueSerde)
		if err != nil {
			return nil, teardownEnclaveDB, fmt.Errorf("failed to create kurtosis value stores: %w", err)
		}

		// We load all the kurtestosis-specific predeclared starlark builtins
		predeclared, err := LoadKurtestosisPredeclared()
		if err != nil {
			return nil, teardownEnclaveDB, err
		}

		// And we create a processor function that merges them with kurtosis predeclared builtins
		processBuiltins := CreateProcessBuiltins(predeclared)

		// We setup a test reporter
		//
		// Besides collecting and formatting the test output (mostly TBD),
		// a reporter is required for correct functioning of the starlarktest assert module
		SetupKurtestosisPredeclared(reporter)

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

		return interpreter, teardownEnclaveDB, err
	}
}
