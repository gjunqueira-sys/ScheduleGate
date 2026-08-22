# Contributing to ScheduleGate

Thank you for contributing to ScheduleGate! This document outlines how to report bugs, submit pull requests, and maintain code quality.

## Reporting Bugs

Open an issue on GitHub with:
- ScheduleGate version (`schedulegate --version`)
- Operating system and architecture
- Sample schedule file (if possible)
- Expected vs. actual behavior

## Pull Requests

1. Fork the repository and create a branch off `main`
2. Make your changes
3. Run `go test ./...` — all tests must pass
4. Submit a pull request with a clear description of the change

## Code Style

- Follow existing patterns in `internal/` and `cmd/`
- Minimalist, direct style — avoid unnecessary comments
- Use descriptive variable names
- Keep functions focused and small

## Testing

- Add tests for new features
- Maintain the 55% minimum coverage gate
- Run `go test ./... -coverprofile=/tmp/coverage.out` to check coverage

## Building

```bash
# macOS (default)
make build

# Windows
make build-windows

# Linux
make build-linux

# All platforms
make build-all
```

## Questions?

Open an issue for discussion before starting large changes.
