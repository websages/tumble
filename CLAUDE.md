# Tumble Project Guidelines

## Application Lifecycle

This project uses a Makefile for all build and run operations. Always use these commands:

| Action | Command | Notes |
|--------|---------|-------|
| Build & Run | `make restart` | Preferred - handles build automatically |
| Build Only | `make build` | Only if you need to compile without running |
| Stop | `make kill` | Always use this to stop the application |
| Run Tests | `make test` | Run the test suite |
| API Tests | `make test-api` | Run API-specific tests |

### Important Constraints

- **Never use direct shell commands** like `docker run`, `go build`, `kill`, `pkill`, or `lsof` to manage the application process
- **Use `make restart`** instead of running `make build` followed by starting the app - restart handles everything
- **If a make command fails**, check the Makefile definition before attempting manual fixes
- **Logs** are in `tumble.log` - use `tail -f tumble.log` to view

## Code Quality

- **No trailing whitespace** - validate with `git diff --check` before committing
- **No Docker usage** without explicitly discussing it first
- **Keep changes minimal** - avoid over-engineering or adding unnecessary features

## Commit Messages

- **Subject line**: Keep under 60 characters
- **Body text**: Wrap at 72 characters per line
- Separate subject from body with a blank line
- Use conventional commit style messages

## API Documentation

When adding, modifying, or removing API routes:
- **Update the OpenAPI spec** at `internal/assets/openapi.json`
- The API docs at `/api/docs` are auto-generated from the OpenAPI spec via Swagger UI
- Include request/response schemas, parameters, and example values in the spec

## Project Structure

This is a Go application. Key directories:
- `internal/` - Internal packages
- `cmd/` - Entry points
- `internal/assets/openapi.json` - OpenAPI specification (update when routes change)
- `Makefile` - Build and lifecycle management
