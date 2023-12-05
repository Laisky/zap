fork from: <https://github.com/uber-go/zap> v1.12.0


Do not set `time.Now` for low level logs.

8x faster than origin zap when you emit huge number of low level logs.
(such as you set log level to INFO then emit DEBUG logs)

![benchmark](https://s3.laisky.com/uploads/2019/02/zap_benchmark.jpeg)

Usage: Replace `"github.com/uber-go/zap"` to `"github.com/Laisky/zap"`.

## New Features

### Hook with fields

New hook `HooksWithFields` is: `func(e zapcore.Entry, fs []zapcore.Field) (err error)`

<<<<<<< HEAD
Example:
=======
```go
logger, _ := zap.NewProduction()
defer logger.Sync()
logger.Info("failed to fetch URL",
  // Structured context as strongly typed Field values.
  zap.String("url", url),
  zap.Int("attempt", 3),
  zap.Duration("backoff", time.Second),
)
```

See the [documentation][doc] and [FAQ](FAQ.md) for more details.

## Performance

For applications that log in the hot path, reflection-based serialization and
string formatting are prohibitively expensive &mdash; they're CPU-intensive
and make many small allocations. Put differently, using `encoding/json` and
`fmt.Fprintf` to log tons of `interface{}`s makes your application slow.

Zap takes a different approach. It includes a reflection-free, zero-allocation
JSON encoder, and the base `Logger` strives to avoid serialization overhead
and allocations wherever possible. By building the high-level `SugaredLogger`
on that foundation, zap lets users *choose* when they need to count every
allocation and when they'd prefer a more familiar, loosely typed API.

As measured by its own [benchmarking suite][], not only is zap more performant
than comparable structured logging packages &mdash; it's also faster than the
standard library. Like all benchmarks, take these with a grain of salt.<sup
id="anchor-versions">[1](#footnote-versions)</sup>

Log a message and 10 fields:

| Package | Time | Time % to zap | Objects Allocated |
| :------ | :--: | :-----------: | :---------------: |
| :zap: zap | 656 ns/op | +0% | 5 allocs/op
| :zap: zap (sugared) | 935 ns/op | +43% | 10 allocs/op
| zerolog | 380 ns/op | -42% | 1 allocs/op
| go-kit | 2249 ns/op | +243% | 57 allocs/op
| slog (LogAttrs) | 2479 ns/op | +278% | 40 allocs/op
| slog | 2481 ns/op | +278% | 42 allocs/op
| apex/log | 9591 ns/op | +1362% | 63 allocs/op
| log15 | 11393 ns/op | +1637% | 75 allocs/op
| logrus | 11654 ns/op | +1677% | 79 allocs/op

Log a message with a logger that already has 10 fields of context:

| Package | Time | Time % to zap | Objects Allocated |
| :------ | :--: | :-----------: | :---------------: |
| :zap: zap | 67 ns/op | +0% | 0 allocs/op
| :zap: zap (sugared) | 84 ns/op | +25% | 1 allocs/op
| zerolog | 35 ns/op | -48% | 0 allocs/op
| slog | 193 ns/op | +188% | 0 allocs/op
| slog (LogAttrs) | 200 ns/op | +199% | 0 allocs/op
| go-kit | 2460 ns/op | +3572% | 56 allocs/op
| log15 | 9038 ns/op | +13390% | 70 allocs/op
| apex/log | 9068 ns/op | +13434% | 53 allocs/op
| logrus | 10521 ns/op | +15603% | 68 allocs/op

Log a static string, without any context or `printf`-style templating:

| Package | Time | Time % to zap | Objects Allocated |
| :------ | :--: | :-----------: | :---------------: |
| :zap: zap | 63 ns/op | +0% | 0 allocs/op
| :zap: zap (sugared) | 81 ns/op | +29% | 1 allocs/op
| zerolog | 32 ns/op | -49% | 0 allocs/op
| standard library | 124 ns/op | +97% | 1 allocs/op
| slog | 196 ns/op | +211% | 0 allocs/op
| slog (LogAttrs) | 200 ns/op | +217% | 0 allocs/op
| go-kit | 213 ns/op | +238% | 9 allocs/op
| apex/log | 771 ns/op | +1124% | 5 allocs/op
| logrus | 1439 ns/op | +2184% | 23 allocs/op
| log15 | 2069 ns/op | +3184% | 20 allocs/op

## Development Status: Stable

All APIs are finalized, and no breaking changes will be made in the 1.x series
of releases. Users of semver-aware dependency management systems should pin
zap to `^1`.

## Contributing

We encourage and support an active, healthy community of contributors &mdash;
including you! Details are in the [contribution guide](CONTRIBUTING.md) and
the [code of conduct](CODE_OF_CONDUCT.md). The zap maintainers keep an eye on
issues and pull requests, but you can also report any negative conduct to
oss-conduct@uber.com. That email list is a private, safe space; even the zap
maintainers don't have access, so don't hesitate to hold us to a high
standard.

<hr>

Released under the [MIT License](LICENSE).

<sup id="footnote-versions">1</sup> In particular, keep in mind that we may be
benchmarking against slightly older versions of other packages. Versions are
pinned in the [benchmarks/go.mod][] file. [↩](#anchor-versions)

[doc-img]: https://pkg.go.dev/badge/github.com/Laisky/zap
[doc]: https://pkg.go.dev/github.com/Laisky/zap
[ci-img]: https://github.com/uber-go/zap/actions/workflows/go.yml/badge.svg
[ci]: https://github.com/uber-go/zap/actions/workflows/go.yml
[cov-img]: https://codecov.io/gh/uber-go/zap/branch/master/graph/badge.svg
[cov]: https://codecov.io/gh/uber-go/zap
[benchmarking suite]: https://github.com/uber-go/zap/tree/master/benchmarks
[benchmarks/go.mod]: https://github.com/uber-go/zap/blob/master/benchmarks/go.mod
>>>>>>> origin/upstream/master

* <https://github.com/Laisky/go-utils/blob/fc5d8c9f5e419e64d4779758f6666dd60cfa6eb4/logger_test.go#L138>
* <https://github.com/Laisky/go-utils/blob/261a79711965d859e6292183b50084e3ab881a12/logger.go#L312>
