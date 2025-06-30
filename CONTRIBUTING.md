# Contributing Guidelines for Agent Mode (Go Projects)

## Agent Mode Principles
- All code should be modular, testable, and easy to extend for automation/agent workflows.
- Prefer automation, scripting, and code generation where possible.
- Document any agent-specific commands, APIs, or automation hooks in this file.

## Go Project Guidelines
- Use idiomatic Go (follow [Effective Go](https://golang.org/doc/effective_go.html)).
- Organize code in packages under `src/`.
- Use `go mod` for dependency management.
- Write unit tests for all new features (`*_test.go`).
- Use `golint`, `go vet`, and `gofmt` before submitting code.
- Document all exported functions and types.
- Use descriptive commit messages.

## Pull Request Process
- Fork the repo and create your branch from `main`.
- Ensure your code builds and passes all tests.
- Open a pull request and describe your changes and agent mode impact.

## Agent Mode Automation
- If you add automation scripts, place them in a `scripts/` directory.
- Document any agent triggers or automation endpoints in this file.

---

For questions, open an issue or contact the maintainers.
