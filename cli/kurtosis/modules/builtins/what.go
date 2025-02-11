package builtins

import (
	"fmt"
	"regexp"

	"github.com/kurtosis-tech/kurtosis/core/server/api_container/server/startosis_engine/kurtosis_instruction/shared_helpers/magic_string_helper"
	"github.com/kurtosis-tech/kurtosis/core/server/api_container/server/startosis_engine/kurtosis_starlark_framework"
	"github.com/kurtosis-tech/kurtosis/core/server/api_container/server/startosis_engine/kurtosis_starlark_framework/builtin_argument"
	"github.com/kurtosis-tech/kurtosis/core/server/api_container/server/startosis_engine/kurtosis_starlark_framework/kurtosis_helper"
	"github.com/kurtosis-tech/kurtosis/core/server/api_container/server/startosis_engine/runtime_value_store"
	"github.com/kurtosis-tech/kurtosis/core/server/api_container/server/startosis_engine/startosis_errors"
	"github.com/sirupsen/logrus"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

const (
	WhatBuiltinName = "what"

	RunArgName = "run"

	// TargetArgName     = "target"
	// MethodNameArgName = "method_name"
)

func NewWhat(
	runtimeValueStore *runtime_value_store.RuntimeValueStore,
) *kurtosis_helper.KurtosisHelper {
	return &kurtosis_helper.KurtosisHelper{
		KurtosisBaseBuiltin: &kurtosis_starlark_framework.KurtosisBaseBuiltin{
			Name: WhatBuiltinName,
			Arguments: []*builtin_argument.BuiltinArgument{
				{
					Name:              RunArgName,
					IsOptional:        false,
					ZeroValueProvider: builtin_argument.ZeroValueProvider[starlark.Value],
					Validator: func(value starlark.Value) *startosis_errors.InterpretationError {
						// TODO
						return nil
					},
				},
			},
		},

		Capabilities: &whatCapabilities{
			runtimeValueStore: runtimeValueStore,
		},
	}
}

type whatCapabilities struct {
	runtimeValueStore *runtime_value_store.RuntimeValueStore
}

func (builtin *whatCapabilities) Interpret(locatorOfModuleInWhichThisBuiltInIsBeingCalled string, arguments *builtin_argument.ArgumentValuesSet) (starlark.Value, *startosis_errors.InterpretationError) {
	runArg, _ := builtin_argument.ExtractArgumentValue[starlark.Value](arguments, RunArgName)
	run, ok := runArg.(*starlarkstruct.Struct)
	if !ok {
		return nil, startosis_errors.NewInterpretationError("oh boi oh boi")
	}

	logrus.Warnf("WHAT ARGS: %v", run.AttrNames())
	logrus.Warnf("WHAT RUN: %s", run)

	code, err := run.Attr("code")
	if !ok {
		return nil, startosis_errors.WrapWithInterpretationError(err, "failed to get code attribute from run result")
	}

	uuidRegexp, err := regexp.Compile(fmt.Sprintf(magic_string_helper.RuntimeValueReplacementPlaceholderFormat, "(.*?)", "code"))
	if err != nil {
		return nil, startosis_errors.WrapWithInterpretationError(err, "failed to create UUID regexp")
	}

	uuidMatches := uuidRegexp.FindStringSubmatch(code.String())
	if len(uuidMatches) != 2 {
		return nil, startosis_errors.WrapWithInterpretationError(err, "failed to get UUID from runtime value")
	}

	uuidMatch := uuidMatches[1]

	logrus.Warnf("WHAT CODE: %s, %s", code, uuidMatch)

	value, err := builtin.runtimeValueStore.GetValue(uuidMatch)
	if err != nil {
		return nil, startosis_errors.WrapWithInterpretationError(err, "failed to get value from the runtime store")
	}

	logrus.Warnf("WHAT\n\n%v", value)

	// builtinModule := builtinValue.(*starlarkstruct.Module)
	// if !builtinModule.Members.Has(methodName) {
	// 	return nil, startosis_errors.NewInterpretationError("method is no good")
	// }

	// // FIXME Create a spy
	// oldMethod := builtinModule.Members[methodName]
	// builtinModule.Members[methodName] = oldMethod

	// logrus.Warnf("WHAT:\n\n%v\n\n%v", builtinModule, builtinModule.Members)

	return starlark.False, nil
}
