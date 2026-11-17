# Tests never shape production

- If a test needs something, the test adapts — mock, fake or helper. Production is never widened to fit it.
- The classic symptom: a mandatory dependency accepting `nil` "so the test passes", which trades a boot failure for silent loss in production. A mandatory dependency is validated by its constructor, which returns an error.
- Same disease: a field or method exported only for a test, an `isTest` flag, a branch only tests run, `time.Sleep` to synchronize.
- A subagent that writes code gets this rule in its prompt.
