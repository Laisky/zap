fork from: <https://github.com/uber-go/zap>


Do not set `time.Now` for low level logs.

10x faster than origin zap when you emit huge number of low level logs.
(such as you set log level to INFO then emit DEBUG logs)

