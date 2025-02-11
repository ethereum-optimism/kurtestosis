mock_test_module = import_module("./mock_test_module.star")

def test_simple(plan):
    mock_run_sh = kurtestosis.mock(plan, "run_sh")

    result = plan.run_sh(
        run = "ls",
    )

    calls = mock_run_sh.calls()
    assert.eq(len(calls), 1)

    assert.eq(calls, [
        struct(args = [], kwargs = { "run": "ls" })
    ])

def test_mock_return_value(plan):
    mock_run_sh = kurtestosis.mock(plan, "run_sh").mock_return_value(42)

    assert.eq(plan.run_sh(run = "pwd"), 42)
    assert.eq(plan.run_sh(run = "ls"), 42)

    calls = mock_run_sh.calls()
    assert.eq(calls, [
        struct(args = [], kwargs = { "run": "pwd" }), 
        struct(args = [], kwargs = { "run": "ls" })
    ])

    return_values = mock_run_sh.return_values()
    assert.eq(return_values, [
        42, 
        42
    ])

def test_mock_return_value_resets_after_test(plan):
    assert.ne(plan.run_sh(run = "ls"), 42)

def test_mock_simple_function(plan):
    mock_target_mock = kurtestosis.mock(mock_test_module, "mock_target").mock_return_value(16)

    assert.eq(mock_test_module.mock_target(), 16)

