# Contributing

## Development

```bash
git clone https://github.com/nickrotondo/lazychez.git
cd lazychez
go build
go test ./...
```

Standard `gofmt` for formatting. No linter config or CI pipeline.

## Testing

```bash
go test ./...              # all tests
go test -v ./...           # verbose
go test ./internal/ui      # single package
go test -run TestFoo ./... # specific test
```

## Releasing

Releases are built with [GoReleaser](https://goreleaser.com/) and automatically update the [Homebrew tap](https://github.com/nickrotondo/homebrew-tap).

```bash
# Tag the release
git tag v0.2.0
git push origin v0.2.0

# Build and publish (creates GitHub release + updates Homebrew cask)
GITHUB_TOKEN=<your-token> goreleaser release --clean
```

The token needs write access to [nickrotondo/homebrew-tap](https://github.com/nickrotondo/homebrew-tap).
