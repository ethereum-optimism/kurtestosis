package builtins

import (
	"fmt"

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
	MockBuiltinName = "mock"

	TargetArgName     = "target"
	MethodNameArgName = "method_name"
)

func NewMock(
	runtimeValueStore *runtime_value_store.RuntimeValueStore,
) *kurtosis_helper.KurtosisHelper {
	return &kurtosis_helper.KurtosisHelper{
		KurtosisBaseBuiltin: &kurtosis_starlark_framework.KurtosisBaseBuiltin{
			Name: MockBuiltinName,
			Arguments: []*builtin_argument.BuiltinArgument{
				{
					Name:              TargetArgName,
					IsOptional:        false,
					ZeroValueProvider: builtin_argument.ZeroValueProvider[starlark.Value],
					Validator: func(value starlark.Value) *startosis_errors.InterpretationError {
						// TODO
						return nil
					},
				},
				{
					Name:              MethodNameArgName,
					IsOptional:        false,
					ZeroValueProvider: builtin_argument.ZeroValueProvider[starlark.String],
					Validator: func(value starlark.Value) *startosis_errors.InterpretationError {
						return builtin_argument.NonEmptyString(value, ServiceNameArgName)
					},
				},
			},
		},

		Capabilities: &mockCapabilities{
			runtimeValueStore: runtimeValueStore,
		},
	}
}

type mockCapabilities struct {
	runtimeValueStore *runtime_value_store.RuntimeValueStore
}

func (builtin *mockCapabilities) Interpret(locatorOfModuleInWhichThisBuiltInIsBeingCalled string, arguments *builtin_argument.ArgumentValuesSet) (starlark.Value, *startosis_errors.InterpretationError) {
	targetArg, _ := builtin_argument.ExtractArgumentValue[starlark.Value](arguments, TargetArgName)
	target, ok := targetArg.(*starlarkstruct.Module)
	if !ok {
		return nil, startosis_errors.NewInterpretationError("oh boi oh boi, %v is not a good module", targetArg)
	}

	methodNameArg, _ := builtin_argument.ExtractArgumentValue[starlark.String](arguments, MethodNameArgName)
	methodName := methodNameArg.GoString()

	if !target.Members.Has(methodName) {
		return nil, startosis_errors.NewInterpretationError("method %s is no good", methodName)
	}

	targetValue := target.Members[methodName]

	logrus.Warnf("MOCK METHOD: %s", targetValue)

	var targetMethod starlarkCallable
	targetMethod, ok = targetValue.(*starlark.Builtin)
	if !ok {
		targetMethod, ok = targetValue.(*starlark.Function)
		if !ok {
			return nil, startosis_errors.NewInterpretationError("oh boi oh boi, %v does not look like a function", targetValue)
		}
	}

	callsArgs := []starlark.Value{}
	returnValues := []starlark.Value{}
	var mockReturnValue starlark.Value
	var mock *starlarkstruct.Module

	target.Members[methodName] = starlark.NewBuiltin(methodName, func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		callArgs := starlark.NewList(args)
		callKwargs, err := kwargsToDict(kwargs)
		if err != nil {
			return nil, startosis_errors.NewInterpretationError("%s: failed to convert kwargs to dict in %s mock: %v", err, MockBuiltinName, methodName)
		}

		callsArgs = append(callsArgs, starlarkstruct.FromStringDict(starlarkstruct.Default, map[string]starlark.Value{
			"args":   callArgs,
			"kwargs": callKwargs,
		}))

		if mockReturnValue != nil {
			returnValues = append(returnValues, mockReturnValue)

			return mockReturnValue, nil
		}

		returnValue, err := targetMethod.CallInternal(thread, args, kwargs)
		if err != nil {
			returnValues = append(returnValues, returnValue)
		}

		return returnValue, err
	})

	modMembers := map[string]starlark.Value{}
	modMembers["calls"] = starlark.NewBuiltin("calls", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		return starlark.NewList(callsArgs), nil
	})
	modMembers["return_values"] = starlark.NewBuiltin("calls", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		return starlark.NewList(returnValues), nil
	})
	modMembers["mock_return_value"] = starlark.NewBuiltin("mock_return_value", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if args.Len() == 0 {
			mockReturnValue = nil
		} else {
			mockReturnValue = args.Index(0)
		}

		return mock, nil
	})

	mock = &starlarkstruct.Module{
		Name:    fmt.Sprintf("mock[%s]", methodName),
		Members: modMembers,
	}

	return mock, nil
}

func kwargsToDict(kwargs []starlark.Tuple) (*starlark.Dict, error) {
	dict := starlark.NewDict(len(kwargs))
	for _, kwarg := range kwargs {
		key, value := kwarg[0], kwarg[1]

		err := dict.SetKey(key, value)
		if err != nil {
			return nil, err
		}
	}

	return dict, nil
}

type starlarkCallable interface {
	CallInternal(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error)
}
