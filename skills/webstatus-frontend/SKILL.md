---
name: webstatus-frontend
description: Use when modifying the frontend SPA, working with TypeScript, Lit web components, Shoelace components, or frontend tests.
---

# webstatus-frontend

This skill provides architectural guidance and conventions for the `frontend/` directory in `webstatus.dev`.

## Architecture & Technology

- **Framework**: Built with **TypeScript**, **Lit**, and **Web Components**.
- **UI Library**: Utilizes **Shoelace** component library.
- **State Management**: Uses **Lit Context** for dependency injection and state management via a service container pattern (`<webstatus-services-container>`).
- **API Interaction**: Communicates with the Go backend using TypeScript types generated from the OpenAPI specification (`make node-openapi`).

## Architecture

For a technical breakdown of the Lit component hierarchy, frontend identity flows, and theming patterns, see [references/architecture.md](references/architecture.md).

## Guidelines (Do's and Don'ts)

- **DO** create new UI elements as custom elements extending Lit's `LitElement`.
- **DO** leverage Shoelace components for common UI patterns.
- **DON'T** introduce other UI frameworks like React or Vue.
- **DO** use Lit Context to access shared services.
- **DON'T** create new global state management solutions.
- **DON'T** render the full page layout (header, sidebar, etc.) inside a page component. Page components should focus on route-specific content; `webstatus-app` provides the shell.
- **DON'T** add generic class names to `shared-css.ts`. **DO** leverage Shadow DOM encapsulation and use composition with slots for reusable layout patterns.
- **DO** write unit tests for all component logic.
- **DO** place application-wide service providers within `WebstatusServicesContainer` to ensure a stable context hierarchy.
- **DO** use specialized child components (the "Context Bridge" pattern) to consume global context if high-level components (like `WebstatusHeader`) don't reliably subscribe to context changes due to slotting or complex rendering lifecycles.
- **DO** use Shoelace semantic CSS variables (e.g., `--sl-color-neutral-0`) for themeable properties to ensure cross-browser inheritance (Firefox/WebKit) without relying on unsupported selectors like `:host-context`.
- **DON'T** directly use Shoelace variables (starting with `--sl-`) in component stylesheets. **DO** use custom variables defined in `_theme-css.ts` (e.g., `--color-background`, `--table-padding`) that act as a project-specific abstraction layer.

## Testing & Linting

- **Test Execution**: `npm run test -w frontend`.
- **Linting**: Run `make node-lint` to run ESLint and Prettier for the frontend code, or `make lint-fix` to attempt auto-fixing. `make style-lint` is also available for CSS.
- **ES Module Testing**: When testing components that use ES module exports directly (e.g. Firebase Auth), use a helper property (e.g. `credentialGetter`) that can be overridden with a Sinon stub.
- **Typing**: Use generic arguments for `querySelector` in tests (e.g. `querySelector<HTMLSlotElement>(...)`) for type safety.

## Theming & Inheritance

- **Global Classes**: The `WebstatusThemeService` toggles the `.sl-theme-dark` class on the `document.documentElement`.
- **Inheritance**: Shoelace variables (e.g., `--sl-color-neutral-0`) automatically switch values based on the root class. Our custom theme variables in `_theme-css.ts` should derive from these semantic Shoelace variables to inherit fixed values across Shadow DOM boundaries consistently.
- **Abstraction Layer**: Components should exclusively use custom variables from `_theme-css.ts`. Mapping these to Shoelace variables should only happen in the central theme file. This ensures that a library or color palette change can be managed in one place.
- **Avoid Unsupported Selectors**: Do not use `:host-context` for theme overrides as it lacks support in Firefox and WebKit. Use root-inherited variables instead.

## Debugging Frontend Tests

If a frontend unit test is timing out or failing mysteriously, Web Test Runner's console output is often unhelpful.

- **Watch Mode**: Instruct the user to run `npm run test:watch -w frontend` in their own terminal.
- **Visual Debugging**: Ask the user to open the provided localhost URL (e.g., `http://localhost:8000/`) in their web browser and inspect the developer console/DOM to see where the test is getting stuck.
- **DON'T** arbitrarily increase the timeout in `web-test-runner.config.mjs` to fix timeout issues. Address the root cause of the hang instead.

## Documentation Updates

When making significant architectural changes to the frontend or introducing new state management patterns:

- Trigger the "Updating the Knowledge Base" prompt in `GEMINI.md`.
- Update `docs/ARCHITECTURE.md` if the system boundaries change.
- Update these very skills files if you introduce new established patterns.
