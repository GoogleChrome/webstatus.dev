# Frontend Architecture & Implementation

This document provides a technical guide for the `frontend/` directory, focusing on Lit components, state management, and user authentication.

## 1. Component Architecture

The frontend is built as a Single Page Application (SPA) using **Lit** for web components.

- **Main Entry**: [frontend/src/static/js/components/webstatus-app.ts](../../../frontend/src/static/js/components/webstatus-app.ts) handles routing and the overall page shell.
- **Styling**: Uses CSS-in-JS patterns specifically designed to support rich aesthetics and high-contrast dark modes.
- **Reusable UI**: Common components (headers, panels, charts) are located in `static/js/components/`.

## 2. Authentication & Identity Flow

The frontend manages user identity via **Firebase Auth** and **GitHub OAuth**.

1.  **Login**: Triggered in the UI; Firebase Auth handles the OAuth handshake with GitHub.
2.  **Token Management**: Upon successful login, Firebase provides a JWT.
3.  **API Requests**: All authorized API calls (e.g., saving a search) attach this JWT as a `Bearer` token in the `Authorization` header.
4.  **User State**: Personalization is driven by the backend `GET /v1/users/me` response, which is never cached.

## 3. Data Integration

- **API Clients**: Standardized Go/TS fetch wrappers communicate with the Backend API.
- **Charts**: Leverages charting libraries to visualize WPT and UMA metrics retrieved from the backend.
- **Search Interaction**: The search bar provides a UI for the complex ANTLR4-based grammar parsed on the backend.
