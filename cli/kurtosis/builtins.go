package kurtosis

import (
	"fmt"
	"kurtestosis/cli/kurtosis/modules"

	"github.com/kurtosis-tech/kurtosis/core/server/api_container/server/startosis_engine"
	"github.com/kurtosis-tech/kurtosis/core/server/api_container/server/startosis_engine/interpretation_time_value_store"
	"github.com/kurtosis-tech/kurtosis/core/server/api_container/server/startosis_engine/runtime_value_store"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarktest"
)

func LoadKurtestosisPredeclared(
	interpretationTimeValueStore *interpretation_time_value_store.InterpretationTimeValueStore,
	runtimeValueStore *runtime_value_store.RuntimeValueStore,
) (starlark.StringDict, error) {
	var err error

	assertPredeclared, err := starlarktest.LoadAssertModule()
	if err != nil {
		return nil, fmt.Errorf("failed to load assert module: %v", err)
	}

	// since assert is a reserved keyword in kurtosis, we provide an alias for it
	expectPredeclared := map[string]starlark.Value{
		"expect": assertPredeclared["assert"],
	}

	kurtestosisPredeclared, err := modules.LoadKurtestosisModule(interpretationTimeValueStore, runtimeValueStore)
	if err != nil {
		return nil, fmt.Errorf("failed to load assert kurtestosis: %v", err)
	}

	return MergeDicts(assertPredeclared, expectPredeclared, kurtestosisPredeclared), nil
}

func CreateProcessBuiltins(extraPredeclared starlark.StringDict) startosis_engine.StartosisInterpreterBuiltinsProcessor {
	return func(thread *starlark.Thread, predeclared starlark.StringDict) starlark.StringDict {
		return MergeDicts(predeclared, extraPredeclared)
	}
}

func SetupKurtestosisPredeclared(reporter starlarktest.Reporter) {
	modules.SetBeforeTestFunction(func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) error {
		starlarktest.SetReporter(thread, reporter)

		return nil
	})
}
