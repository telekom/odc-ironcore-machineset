# Contributing

Thank you for your interest in odc-ironcore-machineset. This guide explains how to propose a
change. By taking part, you agree to follow our [Code of Conduct](CODE_OF_CONDUCT.md).

To report a security vulnerability, follow [SECURITY.md](SECURITY.md) instead of opening an
issue.

## Before You Start

- **Open an issue first** for anything larger than a small fix, so we can agree on the approach
  before you write code.
- Read [`AGENTS.md`](AGENTS.md) for the project layout and the
  [Kubebuilder](https://book.kubebuilder.io) conventions we follow. Some files are generated and
  must not be edited by hand.

## Development Setup

You need the tools listed under **Prerequisites** in the [README](README.md). The repository
also contains a [dev container](.devcontainer/devcontainer.json) with everything preinstalled.

Useful commands:

| Command | What it does |
|---|---|
| `make manifests generate` | Regenerate CRDs, RBAC and DeepCopy code after changing `api/` or markers |
| `make lint-fix` | Fix code style automatically |
| `make test` | Run the unit and controller tests (envtest) |
| `make test-e2e` | Run the end-to-end tests in a new Kind cluster |
| `make check-license` | Check that every Go file has the license header |
| `make help` | List all targets |

## Making a Change

1. **Fork** the repository and create a branch named
   `<type>/<issue-number>-<short-description>`, where `<type>` is one of `feat`, `fix`, `chore`,
   `docs`, `ci` or `test` . Example: `fix/42-reject-empty-template`.
2. **Write tests** for the behaviour you add or fix. We use Go tests with
   [Ginkgo](https://onsi.github.io/ginkgo/) and [Gomega](https://onsi.github.io/gomega/).
3. **Add the license header** to new Go files. `make check-license` shows files without it;
   copy the header from [`hack/boilerplate.go.txt`](hack/boilerplate.go.txt).
4. **Sign your commits** ([GitHub documentation](https://docs.github.com/en/authentication/managing-commit-signature-verification/signing-commits)).
5. **Open a pull request** against `main`. Describe what changes and why, and link the issue.

## Review and Merge

- Every pull request needs an approval from a code owner (see
  [`.github/CODEOWNERS`](.github/CODEOWNERS)) and passing checks: unit tests, e2e tests, lint,
  vulnerability scan and license headers.
- Pull requests are squash-merged, so the pull request title becomes the commit message.
- To publish a release, a codeowner bumps the version in [`VERSION`](VERSION) in a pull request;
  merging it creates the release.

## License

By contributing, you agree that your contributions are licensed under the
[Apache License 2.0](LICENSE).
