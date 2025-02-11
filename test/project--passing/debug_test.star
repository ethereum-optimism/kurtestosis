def test_debug(plan):
    kurtestosis.debug(struct())
    kurtestosis.debug(struct(), message = "hello")
    kurtestosis.debug(message = "hello", value = plan.run_sh)
    kurtestosis.debug(message = "hello", value = struct())
    kurtestosis.debug(message = "hello", value = plan.run_sh)
    kurtestosis.debug(message = "hello", value = {
        "some": "dict"
    })
