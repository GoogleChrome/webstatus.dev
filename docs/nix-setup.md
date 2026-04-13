# Nix Development Environment Setup

This guide explains how to use the Nix development environment for `webstatus.dev`.

## Overview

We are introducing a Nix-based development environment as an alternative to the VS Code DevContainer. This allows developers to have a consistent environment with pinned tool versions without requiring a heavy Docker container for the entire toolchain.

We are currently supporting both workflows side-by-side.

## Prerequisites

1. **Nix**: You must have Nix installed on your host machine. Follow instructions at [nixos.org](https://nixos.org/download.html).
2. **Container Runtime**: A container runtime (like Docker Desktop, Podman, or OrbStack) is still required on your host machine to run Minikube and the Spanner emulator via `make start-local`.

## Getting Started

To enter the development environment, navigate to the project root and run:

```bash
nix develop
make nix-setup
```

This will drop you into a new shell with all the required tools in your `PATH`:

- Go
- Node.js
- OpenJDK
- Terraform, Cloud SDK, Minikube, Skaffold, etc.

You can verify the versions by looking at the output message when you enter the shell.

## Tools Managed by Go

Some tools used in the `Makefile` (like `wrench`, `oapi-codegen`, `golines`, `addlicense`) are not packaged in the Nix flake. Instead, they are managed by Go via `tools/go.mod` using the Go 1.24+ `tool` directive.

Running `make` commands will automatically use `go tool` to resolve and run the correct versions of these tools.

## Wayland vs X11

The Nix environment attempts to detect if your host machine is using Wayland by checking the `$WAYLAND_DISPLAY` environment variable.

- If detected, it sets `MOZ_ENABLE_WAYLAND=1` to ensure Firefox (used in tests) runs natively on Wayland.
- If not detected, it falls back to standard X11 settings.

## Playwright and Browsers

If you need to run E2E tests headfully (where the browser pops up), ensure you have a display server running on your host. For headless tests (default in CI and most local runs), no display is required.

## Verifying Package Versions

You can check what version of a package is provided by the current locked flake without entering the shell by running:

```bash
nix eval nixpkgs#go.version
nix eval nixpkgs#nodejs.version
nix eval nixpkgs#golangci-lint.version
```

## Pinning Specific Tool Versions

If you need to pin a tool like `golangci-lint` to a specific older version (as recommended by its maintainers to avoid using Go tools), you can pull in a specific commit of `nixpkgs` in `flake.nix`.

1. Find the commit hash for the desired version on [nixhub.io](https://nixhub.io).
2. Add a new input in `flake.nix` pointing to that commit. For example, for `golangci-lint` 2.11.0:
   ```nix
   inputs.nixpkgs-lint.url = "github:nixos/nixpkgs/0e6cdd5be64608ef630c2e41f8d51d484468492f";
   ```
3. Use it in the `outputs` function to provide that specific package to your shell.

## Bumping All Tools (Updating the Lockfile)

To update all tools (like Go, Node, Terraform) to their latest versions available in the `nixpkgs-unstable` channel, run:

```bash
nix flake update
```

This will fetch the latest commit from the branch and update `flake.lock`. Note that this may update multiple tools at once, so it is recommended to run tests afterwards to ensure compatibility.

## IDE Integration (VS Code)

To make VS Code aware of the Nix environment automatically in your integrated terminals, you can use `direnv`.

> [!WARNING]
> **Do NOT use the VS Code `direnv` extension (`mkhl.direnv`).** There is a known issue (see [direnv #1307](https://github.com/direnv/direnv/issues/1307)) where the extension loads the environment too early, and the shell startup files subsequently clobber the `PATH`. This results in Nix tools missing in new terminals.

Instead, rely entirely on your host shell's `direnv` hook:

1. **Install `direnv`** on your host machine.
2. **Hook it into your shell**:
   - For **Bash**, add `eval "$(direnv hook bash)"` to your `~/.bashrc`.
   - For **Zsh**, add `eval "$(direnv hook zsh)"` to your `~/.zshrc`.
3. **Create `.envrc`**: Create a file named `.envrc` in the project root with the following content:
   ```bash
   use flake
   ```
   _(Note: This file is ignored in `.gitignore` and should not be checked in)._
4. **Authorize it**: Run `direnv allow` in your terminal.

Now, whenever you open a new integrated terminal in VS Code, your shell will automatically load the Nix environment _after_ initialization, ensuring all tools are available.

## Playwright and Browsers

Playwright manages its own browser binaries and system dependencies.

When you run `make nix-setup`, it runs `npx playwright install` to download browsers, and `npx playwright install-deps webkit` to install system dependencies for WebKit on the host.

We only install dependencies for `webkit` because a full `--with-deps` can fail on newer Linux distributions (like Ubuntu 24.04) due to package name changes (e.g., `libasound2` being replaced by `libasound2t64`).

This ensures that browsers work correctly without needing complex configuration or patching in the Nix flake.

## Playwright E2E Tests and Docker

To guarantee 100% font parity for E2E screenshots across different developer machines (and match CI), you can use a remote Docker browser server.

When you are in the Nix environment, the variable `USE_DOCKER_BROWSER=true` is set by default in `flake.nix`. This tells Playwright to automatically spin up a Docker container (`webstatus-playwright`) and connect to it via WebSocket.

### Known Limitations

> [!IMPORTANT]
> The **"Show browsers"** option in the VS Code Playwright extension is currently **broken** when using the Docker server.
> Because the browser is running inside a container without an XServer, requesting a headed browser will cause tests to fail or not display.

### Workarounds for Debugging

If you need to debug tests and see the browser or interact with it:

1.  **Use UI Mode**: Run `npx playwright test --ui`. This opens an interactive UI on your host that connects to the remote browser and shows snapshots.
2.  **Use Local Browsers**: You can disable the Docker browser temporarily to use your local Nix-patched browsers (which _do_ support showing windows on your host):
    ```bash
    USE_DOCKER_BROWSER=false npx playwright test --headed
    ```
