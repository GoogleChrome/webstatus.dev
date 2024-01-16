# Debugging

This document describes methods to debug the binaries in various environments.

_Note: This document assumes you are using the devcontainer._

## Debug Locally

The skaffold tool comes with a debug command which automatically relaunches the
containers with the appropriate debug tool. [Docs](https://skaffold.dev/docs/workflows/debug/)

- Run: `make debug-local`.
  - _Make sure to stop any existing local servers first._
- Click the `Run and debug` icon on the left side of IDE.
  - You may need to click a "Run and debug button" initially.
- Select the service you want to debug.
  - You can debug different services at the same time.
- Place your breakpoints and exercise those paths
- Ensure to click the disconnect button when finished
