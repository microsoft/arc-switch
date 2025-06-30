# Repository-Wide Best Practices

This document defines the standards and best practices for all projects within this repository. Each subdirectory may also have specific guidelines, but these core principles apply throughout.

## General Go Best Practices

### Code Structure & Organization

- Use the standard Go project layout:
  - `/cmd` - Main applications
  - `/pkg` - Library code that can be used by external applications
  - `/internal` - Private code specific to this repository
  - `/api` - API definitions and specifications
  - `/configs` - Configuration file templates
  - `/test` - Additional test data and tools

- Keep package names concise and meaningful
- One directory = one package

### Code Style

- Follow [Effective Go](https://golang.org/doc/effective_go) guidelines
- Run `gofmt` or `goimports` on all code before committing
- Use meaningful variable and function names that explain purpose
- Keep functions small and focused on a single responsibility
- Limit line length to 100-120 characters for readability

### Documentation

- Document all exported functions, types, and constants
- Write package-level documentation in a `doc.go` file for larger packages
- Use complete sentences with proper punctuation in comments
- Include examples in documentation for non-trivial functionality

### Error Handling

- Return errors rather than using panic
- Wrap errors with context: `fmt.Errorf("doing X: %w", err)`
- Document error return values
- Handle all errors explicitly - don't use `_` to ignore errors

### Testing

- Create tests for all packages
- Use table-driven tests where appropriate
- Aim for >80% test coverage for critical packages
- Use test helpers but keep them in `*_test.go` files
- Create both unit and integration tests

### Dependency Management

- Use Go modules for dependency management
- Minimize external dependencies
- Pin dependency versions explicitly
- Regularly update dependencies for security fixes

### Concurrency

- Use Go's concurrency patterns appropriately
- Prefer channels for communication, mutexes for state
- Always handle goroutine termination
- Avoid goroutine leaks by using context for cancellation

### Project-Specific Tooling

#### Network Tools

- Use standard libraries when possible
- Handle network timeouts gracefully
- Implement retry logic with backoff for transient failures
- Support both IPv4 and IPv6 where applicable

#### Configuration

- Use environment variables for deployment-specific configuration
- Keep sensitive information out of code and config files
- Support configuration reload without restart when feasible
- Validate all configuration values before use

## Git Workflow

- Use feature branches for development
- Create meaningful commit messages that explain "why" not just "what"
- Squash commits before merging to main branch
- Reference issue numbers in commits and PRs

## Code Review Standards

- All code must be reviewed before merging
- Automated tests must pass
- Check for security issues
- Verify error handling is comprehensive
- Ensure documentation is updated

## Project-Specific Standards

Each project directory contains a `PRACTICES.md` file with specific details relevant to that project, which builds upon these repository-wide standards.
