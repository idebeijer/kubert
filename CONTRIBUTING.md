# Contributing to kubert

Thank you for your interest in contributing to kubert! This document provides guidelines and instructions for setting up your development environment and contributing to the project.

## Development Prerequisites

- Go (check [go.mod](go.mod) for minimum version)
- golangci-lint (check [golangci-lint](https://golangci-lint.run/docs/welcome/install/local/) for installation instructions)
- Make (optional, but recommended)

## Getting Started

1. **Clone the repository**

```bash
git clone https://github.com/idebeijer/kubert.git
cd kubert
```

2. **Run from Source**

To run the project locally for example with a custom configuration:

```bash
go run main.go --config config.yaml
```

3. **Build and Test**

You can use `make` to run common tasks:

```bash
make lint
make test
make build
```

4. **Update Documentation**

If you modify CLI commands or add examples, please regenerate the documentation in `./docs`:

```bash
make docs
```

## Commit Messages

We use [Conventional Commits](https://gist.github.com/qoomon/5dfcdf8eec66a051ecd85625518cfd13) for our commit messages. Since we squash commits upon merging, the individual commits in your Pull Request do not necessarily need to follow this convention, but the **PR title** should.

The changelog is automatically generated based on these conventional commits, sorting them by type (e.g., `feat` goes under Features, `fix` under Bug fixes).

### Format

Follow the conventional commit format:

- `feat: add new feature`
- `fix: resolve bug`
- `docs: update documentation`
- `test: add or update tests`
- `refactor: code refactoring`
- `chore: maintenance tasks`

### Breaking Changes

Indicate breaking changes by appending a `!` after the type/scope:

- `feat!: change something`
- `feat(api)!: remove deprecated endpoint`

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
