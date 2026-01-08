# webstatus.dev

[![build](https://github.com/GoogleChrome/webstatus.dev/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/GoogleChrome/webstatus.dev/actions/workflows/ci.yml)
[![GitHub Issues](https://img.shields.io/github/issues/GoogleChrome/webstatus.dev.svg)](https://github.com/anfederico/clairvoyant/issues)
![Contributions welcome](https://img.shields.io/badge/contributions-welcome-orange.svg)

## Overview

[webstatus.dev](https://webstatus.dev) is a tool to monitor and track the status
of all Web Platform features across dimensions that are related to availability
across browsers, and adoption and usage by web developers.

This tool utilizes [workflows](./workflows/) to ingest data from different
public sources such as:

- [Browser Compat Data](https://github.com/mdn/browser-compat-data)
- [Web Platform Tests](https://github.com/web-platform-tests/wpt)
- [Web Features](https://github.com/web-platform-dx/web-features)

The tool serves this data through a Go [backend](./backend/) via an API
described in a [OpenAPI](./openapi/backend/openapi.yaml) document.

The tool provides a [frontend](./frontend/) dashboard written in Typescript to
display the data.

## Search Syntax

webstatus.dev provides a powerful search feature to help you find the
information you need. To learn more about the search syntax and its
capabilities, please refer to our [Search Syntax Guide](./antlr/FeatureSearch.md).

## Get the code

This repository relies heavily on [devcontainers](https://code.visualstudio.com/docs/remote/create-dev-container) to get started.

To continue setting up locally:

```sh
git clone https://github.com/GoogleChrome/webstatus.dev
code webstatus.dev # Opens Visual Studio Code with the webstatus.dev folder.

# While inside Visual Studio Code, start the devcontainer.
# Start it by:
# 1. Opening the Command Palette (via View -> Command Palette)
# 2. Select the option: Dev containers: Rebuild and Reopen in Container
```

### Running the services locally

After getting the code with or without devcontainer, check out the [DEVELOPMENT.md](./DEVELOPMENT.md) for more information to get started and running locally.

### Using Gemini CLI

This devcontainer comes with the [Gemini CLI](https://developers.google.com/gemini-code-assist/docs/gemini-cli) tool. To use it, you can set the `GEMINI_API_KEY` environment variable on your host machine, and it will be automatically configured in the devcontainer. If you do not set it on the host, you will need to authenticate manually after the devcontainer starts.

## Deployment

For project administrators or users interested in deploying their own version,
refer to the [DEPLOYMENT.md](./DEPLOYMENT.md).

## Contributing

We welcome contributions from the community to help make webstatus.dev even
better! There are many ways you can contribute, including:

- Reporting bugs: If you find a bug, please open an issue on GitHub.
- Suggesting enhancements: Have an idea for a new feature or improvement? Open
  an issue to share your suggestion.
- Improving documentation: Help make our documentation clearer and more helpful
  by submitting pull requests with corrections or additions.
- Code contributions: We welcome contributions to our codebase! If you'd like
  to fix a bug or implement a new feature, please submit a pull request.

Before you start contributing, please read our
[CONTRIBUTING.md](./CONTRIBUTING.md) guidelines for details. Additionally,
please review our [Code of Conduct](./CODE_OF_CONDUCT.md).
Thu Jan  8 06:46:07 PM UTC 2026
