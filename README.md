fork from: <https://github.com/uber-go/zap> v1.9.1


Do not set `time.Now` for low level logs.

8x faster than origin zap when you emit huge number of low level logs.
(such as you set log level to INFO then emit DEBUG logs)

![benchmark](https://s3.laisky.com/uploads/2019/02/zap_benchmark.jpeg)

Usage: Replace `"github.com/uber-go/zap"` to `"github.com/Laisky/zap"`.

## New Features

### Hook with fields

New hook func is: `func(e zapcore.Entry, fs []zapcore.Field) (err error)`

Example:

* <https://github.com/Laisky/go-utils/blob/261a79711965d859e6292183b50084e3ab881a12/logger_test.go#L133>
* <https://github.com/Laisky/go-utils/blob/261a79711965d859e6292183b50084e3ab881a12/logger.go#L312>


