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

## Guidelines (Do's and Don'ts)

- **DO** create new UI elements as custom elements extending Lit's `LitElement`.
- **DO** leverage Shoelace components for common UI patterns.
- **DON'T** introduce other UI frameworks like React or Vue.
- **DO** use Lit Context to access shared services.
- **DON'T** create new global state management solutions.
- **DON'T** render the full page layout (header, sidebar, etc.) inside a page component. Page components should focus on route-specific content; `webstatus-app` provides the shell.
- **DON'T** add generic class names to `shared-css.ts`. **DO** leverage Shadow DOM encapsulation and use composition with slots for reusable layout patterns.
- **DO** write unit tests for all component logic.

## Testing & Linting

- **Test Execution**: `npm run test -w frontend`.
- **Linting**: Run `make node-lint` to run ESLint and Prettier for the frontend code, or `make lint-fix` to attempt auto-fixing. `make style-lint` is also available for CSS.
- **ES Module Testing**: When testing components that use ES module exports directly (e.g. Firebase Auth), use a helper property (e.g. `credentialGetter`) that can be overridden with a Sinon stub.
- **Typing**: Use generic arguments for `querySelector` in tests (e.g. `querySelector<HTMLSlotElement>(...)`) for type safety.

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
