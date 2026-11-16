# Air GoLand Demo

A small Go service that receives signed payment webhooks and stores each event once.
It's the demo project for the GoLand blog post about JetBrains Air and the `goland-profiler` skill.
The brief the agents worked from is in [BRIEF.md](BRIEF.md).

Requires Go 1.27.3 or later.

## Branches

| Branch | Contents |
|---|---|
| `main` | The service, with a slow event ID validation left in on purpose. |
| `fixed` | The same service with the fix applied. |

## Run the service

```sh
go run .
```

The server listens on `:8080` and accepts `POST /webhooks/{provider}`.
To send signed requests from GoLand, open `requests/webhooks.http`, select the `dev` environment, and run the requests.

## Run the tests

```sh
go test ./...
```

## Profile the handler

```sh
go test -run '^$' -bench BenchmarkHandler -benchmem ./internal/webhook
```

Compare your working copy with our fix using `git diff origin/fixed`, or view our fix on its own with `git diff origin/main origin/fixed`.
