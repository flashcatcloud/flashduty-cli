# Contributing

Hi there! We're thrilled that you'd like to contribute to Flashduty CLI. Your help is essential for keeping it great.

Contributions to this project are [released](https://docs.github.com/site-policy/github-terms/github-terms-of-service#6-contributions-under-repository-license) to the public under the [project's open source license](LICENSE).

Please note that this project is released with a [Contributor Code of Conduct](CODE_OF_CONDUCT.md). By participating in this project you agree to abide by its terms.

## Prerequisites for running and testing code

These are one-time installations required to test your changes locally as part of the pull request (PR) submission process.

1. Install Go — [download](https://go.dev/doc/install) or [via Homebrew](https://formulae.brew.sh/formula/go). See `go.mod` for the minimum required version.
2. [Install golangci-lint v2](https://golangci-lint.run/welcome/install/).

## Submitting a pull request

1. [Fork](https://github.com/flashcatcloud/flashduty-cli/fork) and clone the repository.
2. Make sure the tests pass on your machine: `go test ./...`
3. Make sure the linter passes on your machine: `golangci-lint run`
4. Create a new branch: `git checkout -b my-branch-name`
5. Make your change, add tests, and make sure the tests and linter still pass.
6. Push to your fork and [submit a pull request](https://github.com/flashcatcloud/flashduty-cli/compare) targeting the `main` branch.
7. Pat yourself on the back and wait for your pull request to be reviewed and merged.

Here are a few things you can do that will increase the likelihood of your pull request being accepted:

- Write tests.
- Keep your change as focused as possible. If there are multiple changes you would like to make that are not dependent upon each other, consider submitting them as separate pull requests.
- Write a [good commit message](https://cbea.ms/git-commit/).

## Resources

- [How to Contribute to Open Source](https://opensource.guide/how-to-contribute/)
- [Using Pull Requests](https://docs.github.com/pull-requests)
- [GitHub Help](https://docs.github.com)
