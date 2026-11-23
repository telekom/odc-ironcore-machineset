# Security Policy

We take the security of odc-ironcore-machineset seriously. This policy follows the
[IronCore security release process](https://github.com/ironcore-dev/.github/blob/main/SECURITY.md),
scaled down to the size of this project.

## Supported Versions

The project is in early development. Only the latest release receives security fixes.

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues, pull requests or
discussions.**

Report them privately through GitHub:

1. Open the [Security tab](https://github.com/telekom/odc-ironcore-machineset/security) of this
   repository.
2. Select **Report a vulnerability**.
3. Describe the issue, the affected version, and the steps to reproduce it.

We will confirm your report, and tell you when we have confirmed or ruled out the issue.

If the vulnerability is in IronCore itself and not in this operator, please report it to the
IronCore project as described in its
[security policy](https://github.com/ironcore-dev/.github/blob/main/SECURITY.md).

## Security Team

The maintainers listed in [`.github/CODEOWNERS`](.github/CODEOWNERS) handle security reports:

- [@takiden](https://github.com/takiden)
- [@shawnsarwar](https://github.com/shawnsarwar)

## Handling a Report

For each confirmed vulnerability, one maintainer leads the fix:

1. **Assess.** Rate the severity with the
   [CVSS calculator](https://www.first.org/cvss/calculator).
2. **Fix privately.** Develop the fix in a
   [temporary private fork](https://docs.github.com/en/code-security/security-advisories/working-with-repository-security-advisories/collaborating-in-a-temporary-private-fork-to-resolve-a-repository-security-vulnerability)
   of the security advisory.
3. **Release.** Merge the fix, publish a release, request a CVE through the GitHub security
   advisory, and publish the advisory with the affected and fixed versions and any steps users
   must take.
4. **Learn.** Write a short, blameless review of how the issue came in and how the response
   went.

If a vulnerability is already public, all steps happen as fast as possible.
