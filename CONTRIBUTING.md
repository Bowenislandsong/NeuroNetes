# Contributing to NeuroNetes

We love your input! We want to make contributing to NeuroNetes as easy and transparent as possible, whether it's:

- Reporting a bug
- Discussing the current state of the code
- Submitting a fix
- Proposing new features
- Becoming a maintainer

## Development Process

We use GitHub to host code, to track issues and feature requests, as well as accept pull requests.

1. Fork the repo and create your branch from `main`.
2. If you've added code that should be tested, add tests.
3. If you've changed APIs, update the documentation.
4. Ensure the test suite passes.
5. Make sure your code lints.
6. Issue that pull request!

## Development Setup

### Prerequisites

- Go 1.21+
- Kubernetes 1.25+
- Docker
- kubectl
- kind (for local testing)

### Local Development

```bash
# Clone the repository
git clone https://github.com/bowenislandsong/neuronetes.git
cd neuronetes

# Install dependencies
make deps

# Generate code
make generate

# Run tests
make test

# Start local development cluster
make dev

# Run controller locally
make run-local
```

### Running Tests

```bash
# Unit tests
make test

# Integration tests
make test-integration

# E2E tests
make test-e2e

# All tests with coverage
make coverage
```

### Code Style

We follow the standard Go code style. Please ensure your code:

- Passes `go fmt`
- Passes `go vet`
- Passes `golangci-lint`
- Has adequate test coverage (>80% for new code)
- Includes comments for exported functions

```bash
# Format code
make fmt

# Run linters
make lint

# Run all verifications
make verify
```

## Pull Request Process

1. Update the README.md with details of changes if applicable.
2. Update the documentation in the `docs/` directory.
3. Add or update tests as needed.
4. Update the CHANGELOG.md with your changes.
5. The PR will be merged once you have the sign-off of at least one maintainer.

### PR Guidelines

- **Title**: Use a clear and descriptive title
- **Description**: Explain what and why, not how
- **Size**: Keep PRs small and focused
- **Tests**: Include tests for new functionality
- **Docs**: Update relevant documentation
- **Commits**: Use meaningful commit messages

### Commit Message Format

We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types:
- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

Examples:
```
feat(scheduler): add GPU topology-aware scheduling

Implements topology-aware bin-packing for multi-GPU workloads.
Considers NVLINK connectivity and NUMA locality.

Closes #123
```

```
fix(autoscaler): prevent scale-down during warm-up period

Fixes race condition where pods were scaled down immediately
after being warmed up.

Fixes #456
```

## Reporting Bugs

We use GitHub issues to track bugs. Report a bug by [opening a new issue](https://github.com/bowenislandsong/neuronetes/issues/new).

### Bug Report Template

**Describe the bug**
A clear and concise description of what the bug is.

**To Reproduce**
Steps to reproduce the behavior:
1. Deploy configuration '...'
2. Run command '....'
3. See error

**Expected behavior**
What you expected to happen.

**Environment:**
- NeuroNetes version: [e.g., v0.1.0]
- Kubernetes version: [e.g., 1.28]
- Cloud provider: [e.g., AWS, GCP, on-prem]
- GPU type: [e.g., A100, H100]

**Logs**
```
Paste relevant logs here
```

## Proposing Features

We use GitHub issues for feature requests. Before creating a feature request:

1. Check if the feature already exists or is being worked on
2. Consider if it aligns with project goals
3. Think about backward compatibility

### Feature Request Template

**Is your feature request related to a problem?**
A clear description of the problem.

**Describe the solution you'd like**
A clear description of what you want to happen.

**Describe alternatives you've considered**
Other solutions or features you've considered.

**Additional context**
Any other context, screenshots, or examples.

## Code Review Process

Maintainers review PRs regularly. We aim to:

- Provide initial feedback within 48 hours
- Complete review within 1 week for simple PRs
- Merge PRs that meet quality standards promptly

### Review Criteria

- Code quality and style
- Test coverage
- Documentation completeness
- Performance impact
- Security considerations
- Breaking changes

## Community

- **Discussions**: Use GitHub Discussions for questions and ideas
- **Issues**: Use GitHub Issues for bugs and feature requests
- **Slack**: Join our Slack workspace (coming soon)

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.

## Questions?

Feel free to open a Discussion or reach out to maintainers!

## Recognition

Contributors will be recognized in:
- CONTRIBUTORS.md file
- Release notes
- Project website (when available)

Thank you for contributing to NeuroNetes! 🚀
