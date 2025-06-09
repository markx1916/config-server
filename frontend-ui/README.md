# Nacos Configuration Management - Frontend UI

This is the Vue 3 + TypeScript + Element Plus frontend for the Nacos Configuration Management tool.

## Prerequisites

- Node.js (v18.x or later recommended)
- npm (or yarn/pnpm)
- Access to the backend API server for this tool.

## Project Setup

1.  **Clone the repository** (if you haven't already). This frontend is typically part of the main project. Navigate to the `frontend-ui` directory:
    ```bash
    cd path/to/your/project/frontend-ui
    ```

2.  **Install Dependencies:**
    The `package.json` file lists all necessary dependencies. Due to potential sandbox limitations with direct `npm install` via automated tools, ensure these dependencies are reflected in your `package.json` and install them in your local environment:
    ```json
    {
      "dependencies": {
        "vue": "^3.4.21",
        "vue-router": "^4.3.0",
        "pinia": "^2.1.7",
        "axios": "^1.6.8",
        "element-plus": "^2.7.0",
        "diff2html": "^3.4.47",
        "diff": "^5.1.0", // For generating diffs
        "vue-codemirror": "^6.0.3",
        "codemirror": "^6.0.1",
        "@codemirror/state": "^6.4.1",
        "@codemirror/view": "^6.26.3",
        "@codemirror/commands": "^6.6.0",
        "@codemirror/language": "^6.10.2",
        "@codemirror/theme-one-dark": "^6.1.2", // Example theme
        "@codemirror/lang-yaml": "^6.1.1",
        "@codemirror/lang-javascript": "^6.2.2",
        "@codemirror/lang-json": "^6.0.1",
        "@codemirror/lang-html": "^6.4.9"
      },
      "devDependencies": {
        "@vitejs/plugin-vue": "^5.0.4",
        "typescript": "^5.2.2",
        "vue-tsc": "^2.0.6",
        "vite": "^5.2.0",
        "sass": "^1.71.1"
      }
    }
    ```
    In your local environment, run:
    ```bash
    npm install
    ```

3.  **Configure Backend API URL:**
    The frontend makes API calls to the backend server. The base URL for these API calls is configured in `frontend-ui/vite.config.ts` under the `server.proxy` section and in `frontend-ui/src/services/api.ts` (`API_BASE_URL`).

    -   **Development (Vite Dev Server):**
        The `vite.config.ts` includes a proxy for `/api` requests to `http://localhost:8080` (the default backend port).
        ```typescript
        // frontend-ui/vite.config.ts
        // ...
        server: {
          port: 3000, // Frontend dev server port
          proxy: {
            '/api': {
              target: 'http://localhost:8080', // Backend API URL
              changeOrigin: true,
            },
          },
        },
        // ...
        ```
        Ensure the `target` in the proxy configuration matches where your backend server is running.

    -   **Production Build:**
        When building for production, the `API_BASE_URL` in `src/services/api.ts` will be used. By default, it's `/api`. This means your production deployment should serve the frontend and backend in a way that API requests to `/api/...` are routed to the backend server (e.g., using Nginx reverse proxy).
        You can also configure this via Vite environment variables (e.g., `VITE_API_BASE_URL`). Create a `.env.production` file in the `frontend-ui` directory:
        ```
        VITE_API_BASE_URL=https://your-backend.example.com/api
        # Or for same-domain deployment:
        # VITE_API_BASE_URL=/api
        ```

## Running the Development Server

1.  Ensure the backend server is running and accessible.
2.  Start the Vite development server:
    ```bash
    npm run dev
    ```
    This will typically start the frontend on `http://localhost:3000`.

## Building for Production

1.  Build the application:
    ```bash
    npm run build
    ```
    This will create a `dist` directory in `frontend-ui` containing the static assets.

2.  **Deployment:**
    The contents of the `dist` folder can be served by any static web server (like Nginx, Apache, or cloud storage services). Ensure your server is configured to handle SPA routing (redirect all non-asset requests to `index.html`).
    If serving alongside the backend on the same domain, configure your reverse proxy (e.g., Nginx) to serve the frontend static files and proxy API requests to the backend.

## Key Features Implemented

-   **Nacos Instance Management:** List, add, edit, delete, and test connections to Nacos server instances.
-   **Configuration Listing:** View configurations based on selected Nacos instance and namespace.
-   **Configuration Editor:** Create and edit configurations with a CodeMirror-based editor, supporting various content types (text, JSON, YAML, HTML).
-   **Diff View:** Compare local configuration content with the version stored in Nacos using `diff` and `diff2html`.
-   **Publishing:** Publish local configurations to Nacos (Full publish implemented).
-   **History & Rollback:** View configuration history and rollback to previous versions.
-   **Publish Records:** View a log of publish attempts and their statuses.
-   **Responsive UI:** Using Element Plus components for a clean and functional user interface.
-   **State Management:** Pinia for managing shared state like selected Nacos instance and namespaces.

## Project Structure Overview

-   `src/App.vue`: Main application layout with sidebar and content area.
-   `src/main.ts`: Vue app initialization, plugins (Vue Router, Pinia, Element Plus).
-   `src/router/index.ts`: Vue Router configuration and route definitions.
-   `src/services/api.ts`: Axios instance and API communication functions.
-   `src/store/index.ts`: Pinia state management stores.
-   `src/types/index.ts`: TypeScript interfaces for data models.
-   `src/views/`: Page-level components for each route.
-   `src/components/`: Reusable UI components (currently placeholders, views are self-contained).
-   `src/assets/styles/main.scss`: Global SCSS styles.
-   `public/`: Static assets copied directly to the build output (e.g., `favicon.ico`, `logo.svg`).
-   `vite.config.ts`: Vite build and development server configuration.
-   `package.json`: Project dependencies and scripts.

## Available Scripts

-   `npm run dev`: Starts the development server.
-   `npm run build`: Builds the application for production using `vue-tsc` for type checking and Vite for bundling.
-   `npm run preview`: Serves the production build locally for preview.
```
