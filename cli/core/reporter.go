package core

import (
	"github.com/sirupsen/logrus"
	"go.starlark.net/starlarktest"
)

type StarlarkTestError = []interface{}

var (
	_ StarlarkReporter = (*StarlarkTestSuiteReporter)(nil)
	_ StarlarkReporter = (*StarlarkTestFileReporter)(nil)
	_ StarlarkReporter = (*StarlarkTestFunctionReporter)(nil)
)

type StarlarkReporter interface {
	starlarktest.Reporter

	Before()
	After()
	Success() bool
	Nest(child StarlarkReporter)
}

type StarlarkReporterBase struct {
	errors   []interface{}
	children []StarlarkReporter
}

func (reporter *StarlarkReporterBase) Error(args ...interface{}) {
	reporter.errors = append(reporter.errors, args)
}

func (reporter *StarlarkReporterBase) Before() {}

func (reporter *StarlarkReporterBase) After() {}

func (reporter *StarlarkReporterBase) Success() bool {
	for _, child := range reporter.children {
		if !child.Success() {
			return false
		}
	}

	return true
}

func (reporter *StarlarkReporterBase) Nest(child StarlarkReporter) {
	reporter.children = append(reporter.children, child)
}

type StarlarkTestSuiteReporter struct {
	StarlarkReporterBase
	Logger  *logrus.Logger
	Project *KurtestosisProject
}

func CreateStarlarkTestSuiteReporter(project *KurtestosisProject, logger *logrus.Logger) *StarlarkTestSuiteReporter {
	return &StarlarkTestSuiteReporter{
		Logger:  logger,
		Project: project,
	}
}

type StarlarkTestFileReporter struct {
	StarlarkReporterBase
	Logger   *logrus.Logger
	TestFile *TestFile
}

func (reporter *StarlarkTestFileReporter) Before() {
	reporter.Logger.Infof("SUITE %s", reporter.TestFile)
}

func CreateStarlarkTestFileReporter(testFile *TestFile, logger *logrus.Logger) *StarlarkTestFileReporter {
	return &StarlarkTestFileReporter{
		Logger:   logger,
		TestFile: testFile,
	}
}

type StarlarkTestFunctionReporter struct {
	StarlarkReporterBase
	Logger       *logrus.Logger
	TestFunction *TestFunction
}

func (reporter *StarlarkTestFunctionReporter) Before() {
	reporter.Logger.Infof("\tRUN %s", reporter.TestFunction.Name)
}

func (reporter *StarlarkTestFunctionReporter) After() {
	if reporter.Success() {
		reporter.Logger.Infof("\tPASS %s", reporter.TestFunction.Name)
	} else {
		reporter.Logger.Infof("\tFAIL %s:\n================================================\n%v\n================================================", reporter.TestFunction.Name, reporter.errors)
	}
}

func (reporter *StarlarkTestFunctionReporter) Success() bool {
	return len(reporter.errors) == 0
}

func CreateStarlarkTestFunctionReporter(testFunction *TestFunction, logger *logrus.Logger) *StarlarkTestFunctionReporter {
	return &StarlarkTestFunctionReporter{
		Logger:       logger,
		TestFunction: testFunction,
	}
}

// type TestSuiteSummary struct {
// 	Project   *KurtestosisProject
// 	summaries []TestFileSummary
// }

// func NewTestSuiteSummary(project *KurtestosisProject) *TestSuiteSummary {
// 	return &TestSuiteSummary{
// 		Project: project,
// 	}
// }

// func (summary *TestSuiteSummary) Append(testFileSummary *TestFileSummary) {
// 	summary.summaries = append(summary.summaries, *testFileSummary)
// }

// func (summary *TestSuiteSummary) Summaries() []TestFileSummary {
// 	return summary.summaries
// }

// func (summary *TestSuiteSummary) Success() bool {
// 	for _, testFileSummary := range summary.summaries {
// 		if !testFileSummary.Success() {
// 			return false
// 		}
// 	}

// 	return true
// }

// type TestFileSummary struct {
// 	TestFile  *TestFile
// 	summaries []TestFunctionSummary
// }

// func (summary *TestFileSummary) Summaries() []TestFunctionSummary {
// 	return summary.summaries
// }

// func (summary *TestFileSummary) Append(testFunctionSummary *TestFunctionSummary) {
// 	summary.summaries = append(summary.summaries, *testFunctionSummary)
// }

// func (summary *TestFileSummary) Success() bool {
// 	for _, testFunctionSummary := range summary.summaries {
// 		if !testFunctionSummary.Success() {
// 			return false
// 		}
// 	}

// 	return true
// }

// type TestFunctionSummary struct {
// 	TestFunction *TestFunction
// 	errors       []StarlarkTestError
// }

// func (summary *TestFunctionSummary) Errors() []StarlarkTestError {
// 	return summary.errors
// }

// func (summary *TestFunctionSummary) Success() bool {
// 	return len(summary.errors) == 0
// }

// type TestReporter struct {
// 	TestFunction *TestFunction
// 	errors       []StarlarkTestError
// }

// func (reporter *TestReporter) Error(args ...interface{}) {
// 	reporter.errors = append(reporter.errors, args)
// }

// func (reporter *TestReporter) Summary() *TestFunctionSummary {
// 	return &TestFunctionSummary{
// 		TestFunction: reporter.TestFunction,
// 		errors:       reporter.errors,
// 	}
// }

// func NewTestReporter(testFunction *TestFunction) *TestReporter {
// 	return &TestReporter{
// 		TestFunction: testFunction,
// 	}
// }

// func NewTestFileSummary(testFile *TestFile) *TestFileSummary {
// 	return &TestFileSummary{
// 		TestFile: testFile,
// 	}
// }
