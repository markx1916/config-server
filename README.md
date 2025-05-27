# Nacos Configuration Center (Vue3 + Go)

This project is a web-based tool for managing configurations stored in Nacos. It provides a user interface built with Vue3 and Element Plus, and a backend API service built with Golang (Gin framework). The backend uses a local MySQL database for versioning, history tracking, and controlled publishing to Nacos instances.

## Features

**Backend:**
- Manage multiple Nacos server instances.
- Manage configurations (Data ID, Group, Namespace) with local versioning.
- Publish configurations to Nacos (full or grayscale).
- View configuration history and rollback to previous versions.
- Track publish records.
- Basic OpenTelemetry tracing for HTTP requests.

**Frontend:**
- CRUD operations for Nacos server instances.
- List, create, edit configurations associated with a Nacos instance.
- View configuration content, save history, and deployment history in a tabbed interface.
- Diff view to see changes before saving or compare with previous versions.
- Publish configurations (Full/Gray) to Nacos.
- Rollback to a previous version of a configuration (locally).
- User-friendly interface using Element Plus components.

## Project Structure (Current Monorepo-like Setup at `/app`)

- **Backend (Go):**
    - `main.go`: Main application entry point for the backend.
    - `go.mod`, `go.sum`: Go module files.
    - `config/`: Configuration loading (Viper) and `config.yaml`.
    - `db/`: Database initialization (GORM) and `schema.sql`.
    - `handlers/`: Gin API request handlers.
    - `models/`: GORM database models.
    - `routes/`: Gin router setup.
    - `services/`: Business logic, including Nacos SDK interaction.
    - `utils/`: Utility functions (e.g., logger).
    - `config.yaml.example`: Example backend configuration file.
- **Frontend (Vue3):**
    - `index.html`: Main HTML entry point for the frontend.
    - `package.json`: Frontend dependencies and scripts.
    - `vite.config.ts`: Vite configuration, including dev server proxy.
    - `tsconfig.json`, `tsconfig.node.json`: TypeScript configurations.
    - `src/`: Frontend source code.
        - `main.ts`: Vue app initialization (Vue, Element Plus, Pinia, Router).
        - `App.vue`: Main Vue application shell with navigation.
        - `router/`: Vue Router setup.
        - `store/`: Pinia state management stores (`nacosStore`, `configStore`).
        - `services/api.ts`: Axios setup and API call definitions.
        - `views/`: Page-level components (Nacos Instance Management, Config List, Config Detail).
        - `components/`: Reusable UI components (history tabs, NotFound).
- `schema.sql`: MySQL database schema.
- `Dockerfile`: For building the backend Docker image (frontend can be built separately or served via backend).

## Prerequisites

- Go 1.23 or later (due to OpenTelemetry dependency, check `go.mod`)
- Node.js 18.x or later (for frontend)
- npm (or yarn/pnpm)
- MySQL 5.7 (or compatible)
- Access to one or more Nacos server instances

## Setup and Running

### 1. Backend Setup

a.  **Database:**
    *   Ensure you have a MySQL instance running and accessible.
    *   Create a database, e.g., `nacos_config_center_db`.
    *   The backend will attempt to auto-migrate the schema defined in `schema.sql` upon first run. You can also apply it manually.

b.  **Configuration (`/app/config.yaml`):**
    *   Copy `config.yaml.example` to `config.yaml`.
    *   Edit `config.yaml` with your specific settings:
        *   `SERVER_PORT`: Port for the backend API server (e.g., "8080").
        *   `DATABASE.DSN`: MySQL Data Source Name (e.g., "user:pass@tcp(127.0.0.1:3306)/nacos_config_center_db?charset=utf8mb4&parseTime=True&loc=Local").
        *   `NACOS.*`: Default Nacos client settings for the `nacos_service.go` if not connecting via a managed instance.
    *   Alternatively, use environment variables prefixed with `NCT_` (e.g., `NCT_SERVER_PORT=8080`, `NCT_DATABASE_DSN="your_dsn"`). Environment variables override `config.yaml` values.

c.  **Run Backend:**
    Open a terminal in the project root (`/app`):
    ```bash
    # Install/update Go dependencies
    go mod tidy

    # Run the backend server
    go run main.go
    ```
    The backend server should start, typically on `http://localhost:8080` (or as configured in `config.yaml`).

### 2. Frontend Setup

a.  **Run Frontend Dev Server:**
    Open another terminal in the project root (`/app`):
    ```bash
    # Install frontend dependencies
    npm install

    # Run the frontend development server
    npm run dev
    ```
    The frontend development server will start, typically on `http://localhost:3000` (as configured in `vite.config.ts`, which was previously set to 3000, but Vite's default is 5173. The current `vite.config.ts` has port 3000).

b.  **Accessing the Application:**
    Open your browser and navigate to `http://localhost:3000` (or the port shown by `npm run dev`).

    The `vite.config.ts` is configured to proxy API requests from `/api` on the frontend dev server to `http://localhost:8080/api` on the backend.

### 3. Running with Docker (Backend Only Example)

A `Dockerfile` is provided for building the backend as a Docker image.
```bash
# Build the Docker image for the backend
docker build -t nacos-config-center-backend .

# Run the Docker container (example)
# Ensure config.yaml is correctly mounted or environment variables are set
docker run -d -p 8080:8080 --name nacos-center-backend \
  -v $(pwd)/config.yaml:/app/config.yaml \
  nacos-config-center-backend
```
For a full Dockerized setup including the frontend, you would typically build the frontend into static assets and either serve them from the Go backend or use a multi-container Docker Compose setup.

## API Endpoints (Summary)

(Refer to `routes/routes.go` for exact definitions and `handlers/` for implementation.)

**Nacos Instances:** (`/api/nacos/instances`)
- `POST /`: Create an instance.
- `GET /`: List all instances.
- `GET /:id`: Get a specific instance.
- `PUT /:id`: Update an instance.
- `DELETE /:id`: Delete an instance.

**Configurations:** (`/api/nacos/configs`)
- `POST /`: Create a configuration.
- `GET /`: List configurations (requires `nacos_instance_id` query param).
- `GET /:id`: Get a specific configuration.
- `PUT /:id`: Update a configuration (creates new version history).
- `GET /:id/diff`: Get diff between current and a previous version.
- `POST /:id/publish/:type`: Publish to Nacos (`type` is `gray` or `full`).
- `GET /:id/history`: Get save history of a configuration.
- `POST /:id/rollback/:history_id`: Rollback to a historical version.
- `GET /:id/deployments`: Get Nacos deployment history for a configuration.

**Health Check:**
- `GET /health` (backend)

## Logging

- **Backend**: Structured JSON logs via Zap logger. HTTP request logs include method, path, status, latency, client IP. Log level can be configured via `LOG_LEVEL` environment variable (e.g., `DEBUG`, `INFO`).
- **Frontend**: Uses `console.log` for general logging and `ElMessage` for user feedback.

## Potential Enhancements / TODO

- **Password Encryption:** Store Nacos instance passwords securely in the backend database.
- **User Authentication & Authorization:** Implement robust API authentication and role-based access control for both frontend and backend.
- **Real Feishu Notifications:** Currently a placeholder; integrate a Feishu client library.
- **Advanced Nacos Features:** Deeper integration with Nacos features like tags, listening for remote changes.
- **Comprehensive Testing:** Add unit and integration tests for both frontend and backend.
- **Deployment Strategy:** Define a clear production deployment strategy (e.g., build frontend into static assets served by Go, or separate Docker containers with Nginx).
- **Directory Structure:** Consider separating frontend and backend into distinct subdirectories for better organization in larger projects.
```
