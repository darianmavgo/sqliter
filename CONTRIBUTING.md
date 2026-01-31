# Contributing to SQLiter

First off, thanks for taking the time to contribute! 🎉

The following is a set of guidelines for contributing to SQLiter. These are just guidelines, not rules. Use your best judgment and feel free to propose changes to this document in a pull request.

## How Can I Contribute?

### Reporting Bugs

This section guides you through submitting a bug report. Following these guidelines helps maintainers and the community understand your report, reproduce the behavior, and find related reports.

- **Use a clear and descriptive title** for the issue to identify the problem.
- **Describe the exact steps to reproduce the problem** in as much detail as possible.
- **Provide specific examples** to demonstrate the steps.
- **Describe the behavior you observed** after following the steps and point out what exactly is the problem with that behavior.
- **Explain which behavior you expected to see instead** and why.

### Suggesting Enhancements

This section guides you through submitting an enhancement suggestion, including completely new features and minor improvements to existing functionality.

- **Use a clear and descriptive title** for the issue.
- **Provide a step-by-step description of the suggested enhancement** in as much detail as possible.
- **Provide specific examples** to demonstrate the steps.
- **Describe the current behavior** and **explain which behavior you expected to see instead** and why.

### Pull Requests

1.  Fork the repo and create your branch from `main`.
2.  If you've added code that should be tested, add tests.
3.  If you've changed APIs, update the documentation.
4.  Ensure the test suite passes.
5.  Make sure your code lints.

## Development Setup

1.  **Backend**:
    ```bash
    go mod download
    ```

2.  **Frontend**:
    ```bash
    cd react-client
    npm install
    ```

3.  **Build**:
    ```bash
    ./react_go_build.sh
    ```

## Styleguides

### Go Styleguide

- Use `gofmt` to format your code.
- Follow [Effective Go](https://golang.org/doc/effective_go.html).

### JavaScript/React Styleguide

- Use standard ESLint configuration provided in the project.
