# CI testing policy (2026-10-08)

At the maintainer's request, automatic pre-merge testing is repository-wide
gofmt and eleven explicitly named unit tests for logging level conversion and
encoder registration/error handling. This limited gate runs Go 1.27.1 once
without coverage, race instrumentation, performance, fuzz, environment tests
or auxiliary-module dependency installation. Discovery and execution must
include every allowlisted test exactly once; skips, errors, missing tests,
invalid JSON and timeouts fail. Raw command exits and timings are uploaded on
both failure and success. Build caching is enabled; test-result caching is off.

```sh
python3 -m unittest discover -s .scripts -p test_fast_ci.py -v
python3 .scripts/fast_ci.py --evidence /tmp/zap-fast-ci
```

The previous complete coverage matrix (Go 1.26 and 1.27), Codecov upload and
full lint job remain intact in `.github/workflows/go-full.yml`, available through
Actions > Go full qualification > Run workflow against the candidate branch.
The existing `make vulncheck` source scan remains automatic in `security.yml`;
its runtime is separate from the fast unit budget. No security scanner is
removed. Full qualification remains mandatory on local dev/staging for relevant
changes; this schedule amendment supersedes prior automatic-CI descriptions.

```sh
make lint
make test
make cover
make bench BENCH=.
```

Run compatibility qualification on Go 1.26 and 1.27. The Makefile retains all
auxiliary module, benchmark, race and coverage commands and assertions.
This change neither publishes/deploys artifacts nor alters secrets, credentials,
repository security or branch protection settings. A five-minute quick-job
timeout limits runaway tests; measured clean/cached runtime is recorded in the
PR separately from toolchain download and GitHub runner setup.
