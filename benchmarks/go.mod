module github.com/Laisky/zap/benchmarks

go 1.21

replace github.com/Laisky/zap => ../

require (
	github.com/Laisky/zap v1.23.0
	github.com/apex/log v1.9.0
	github.com/go-kit/log v0.2.1
	github.com/rs/zerolog v1.30.0
	github.com/sirupsen/logrus v1.9.3
	go.uber.org/multierr v1.11.0
	gopkg.in/inconshreveable/log15.v2 v2.16.0
)

require (
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	golang.org/x/lint v0.0.0-20190930215403-16217165b5de // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/term v0.13.0 // indirect
	golang.org/x/tools v0.1.5 // indirect
)
