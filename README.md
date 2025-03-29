# GoScry

<p align="center">
  <img src="media/logo.png" alt="GoScry Logo" width="200">
</p>

GoScry is a server application written in Go that acts as a bridge between a controlling system (like an LLM or script) and a web browser. It uses the Chrome DevTools Protocol (CDP) to interact with websites based on tasks submitted via its API. GoScry can perform actions like navigation, clicking, typing, handling authentication (with hooks for 2FA), and extracting DOM content. Results and status updates can be reported back via webhooks using the Model Context Protocol (MCP) format.

## Features

* **Remote Browser Control:** Uses CDP (via `chromedp`) to control headless or headed Chrome/Chromium instances.
* **Task-Based API:** Submit sequences of browser actions (navigate, click, type, wait, get DOM, screenshot, etc.) via a simple JSON API.
* **Authentication Handling:** Supports basic username/password login sequences within tasks.
* **2FA Support (Manual Hook):** Detects potential 2FA prompts and signals back via API/callback, allowing an external system or user to provide the code to continue the task.
* **DOM Extraction:** Retrieve full HTML, text content, or a simplified version of the DOM.
* **MCP Output:** Formats asynchronous results/status updates (e.g., via callbacks) according to the Model Context Protocol (spec 2025-03-26) for clear, structured context reporting.
* **Configurable:** Manage server port, browser settings, logging, and security via a YAML file or environment variables.

## Architecture Diagram

```mermaid
sequenceDiagram
    participant Client as LLM System / API Client
    participant GS as GoScry Server (API)
    participant TM as Task Manager
    participant BM as Browser Manager
    participant CDP as Chrome (via CDP)
    participant Site as Target Website

    Client->>+GS: POST /api/v1/tasks (Task JSON)
    GS->>+TM: SubmitTask(task)
    TM-->>GS: TaskID
    GS-->>-Client: 202 Accepted (TaskID)
    Note right of TM: Task Execution starts async
    TM->>BM: ExecuteActions(actions)
    BM->>+CDP: Run CDP Commands (Navigate, Click, etc.)
    CDP->>+Site: HTTP Request
    Site-->>-CDP: HTTP Response (HTML, etc.)
    CDP-->>-BM: Action Result / DOM State
    BM-->>TM: Action Completed / Error
    alt 2FA Required (e.g., after login action)
        TM->>BM: ExecuteAction(detect2FAPrompt)
        BM->>CDP: Check page state for 2FA indicators
        CDP->>BM: Presence result
        BM->>TM: Prompt detected/not detected
        opt Prompt Detected
            TM->>TM: Update Task Status (WaitingFor2FA)
            TM-->>Client: Notify Callback (MCP 2FA Request)
            Note over Client, TM: Client retrieves code externally
            Client->>+GS: POST /api/v1/tasks/{id}/2fa (Code)
            GS->>+TM: Provide2FACode(id, code)
            TM->>TM: Signal/Resume Task Execution
            Note right of TM: Next action types the 2FA code
            TM->>BM: ExecuteActions(type 2FA code, submit)
            BM->>+CDP: Type Code, Submit
            CDP->>+Site: Verify 2FA
            Site-->>-CDP: Login Success/Failure
            CDP-->>-BM: Result
            BM-->>TM: Action Completed
        end
    end
    TM->>TM: Process Final Result / Format MCP
    TM-->>Client: Notify Callback (MCP Result/Status)
    TM->>TM: Update Task Status (Completed/Failed)
    Note over TM: Task finished execution.
```

## Package Structure

* **cmd/goscry:** Main application entry point
* **internal/taskstypes:** Core data types shared across packages
* **internal/tasks:** Task management and execution
* **internal/browser:** Browser control and CDP interactions
* **internal/server:** HTTP API handlers
* **internal/config:** Configuration handling
* **internal/dom:** DOM processing utilities

## Prerequisites

* **Go:** Version 1.21 or later.
* **Chrome / Chromium:** A compatible version installed on the system where GoScry will run. Ensure the browser executable is in the system PATH or provide the path in the configuration.

## Installation

1.  Clone the repository:
    ```bash
    git clone https://github.com/copyleftdev/goscry.git
    cd goscry
    ```
2.  Build the executable:
    ```bash
    go build -o goscry ./cmd/goscry/
    ```
    This will create the `goscry` binary in the project root.

## Configuration

GoScry is configured via a `config.yaml` file or environment variables.

1.  Copy the example configuration:
    ```bash
    cp config.example.yaml config.yaml
    ```
2.  Edit `config.yaml` to suit your environment:
    * `server.port`: Port the API server listens on.
    * `browser.executablePath`: Absolute path to the Chrome/Chromium executable (leave empty to attempt auto-detect).
    * `browser.headless`: `true` to run headless, `false` for headed mode.
    * `browser.userDataDir`: Path to a persistent user profile directory (optional, creates temporary profile if empty).
    * `browser.maxSessions`: Maximum concurrent browser instances.
    * `log.level`: Logging level (`debug`, `info`, `warn`, `error`).
    * `security.allowedOrigins`: List of origins allowed for CORS requests. Use specific domains in production instead of `*`.
    * `security.apiKey`: A secret key required for API access (set via `GOSCRY_SECURITY_APIKEY` environment variable for better security).

Environment variables override file settings. They are prefixed with `GOSCRY_` and use underscores instead of dots (e.g., `GOSCRY_SERVER_PORT=9090`, `GOSCRY_SECURITY_APIKEY=your-secret-key`).

## Running the Server

```bash
./goscry -config config.yaml
```

Or, if using environment variables primarily:

```bash
export GOSCRY_SECURITY_APIKEY="your-secret-key"
# export GOSCRY_BROWSER_EXECUTABLEPATH="/path/to/chrome" # Optional
./goscry
```

The server will start and log output to the console based on the configured log level.

## API Usage

The API listens on the configured port (default 8080) under the `/api/v1` path prefix. Authentication via `X-API-Key` or `Authorization: Bearer <key>` header is required if `security.apiKey` is set.

### Endpoints

* **`POST /api/v1/tasks`**: Submit a new browser task.
    * **Request Body:** `SubmitTaskRequest` JSON (see `internal/server/handlers.go`). Includes `actions`, optional `credentials`, `two_factor_auth` info, and `callback_url`.
    * **Response (Success):** `202 Accepted` with `SubmitTaskResponse` JSON containing the `task_id`.
    * **Response (Error):** `400 Bad Request`, `401 Unauthorized`, `403 Forbidden`, `500 Internal Server Error`.

* **`GET /api/v1/tasks/{taskID}`**: Get the current status and result of a task.
    * **URL Parameter:** `taskID` (UUID string).
    * **Response (Success):** `200 OK` with `Task` JSON (see `internal/tasks/task.go`).
    * **Response (Error):** `400 Bad Request`, `401 Unauthorized`, `403 Forbidden`, `404 Not Found`, `500 Internal Server Error`.

* **`POST /api/v1/tasks/{taskID}/2fa`**: Provide a 2FA code for a task waiting for it.
    * **URL Parameter:** `taskID` (UUID string).
    * **Request Body:** `Provide2FACodeRequest` JSON (e.g., `{"code": "123456"}`).
    * **Response (Success):** `200 OK` with simple success message.
    * **Response (Error):** `400 Bad Request`, `401 Unauthorized`, `403 Forbidden`, `404 Not Found`, `409 Conflict` (if task not waiting), `408 Request Timeout` (if task timed out waiting), `500 Internal Server Error`.

### Action Types

The `actions` array in the submit request defines the steps:

| Type              | Description                                                                 | `selector` Used | `value` Used                                                               | `format` Used               |
| :---------------- | :-------------------------------------------------------------------------- | :-------------- | :------------------------------------------------------------------------- | :-------------------------- |
| `navigate`        | Navigates the browser to a URL.                                             | No              | URL string                                                                 | No                          |
| `wait_visible`    | Waits for an element matching the selector to become visible.               | Yes             | Optional duration (e.g., "5s", default "30s")                              | No                          |
| `wait_hidden`     | Waits for an element matching the selector to become hidden.                | Yes             | Optional duration (e.g., "5s", default "30s")                              | No                          |
| `wait_delay`      | Pauses execution for a specified duration.                                  | No              | Duration string (e.g., "2s", "500ms")                                      | No                          |
| `click`           | Waits for an element to be visible and clicks it.                           | Yes             | No                                                                         | No                          |
| `type`            | Types text into an element. Use `{{task.tfa_code}}` for 2FA code injection. | Yes             | Text string, or `{{task.tfa_code}}`                                        | No                          |
| `select`          | Selects an option within a `<select>` element by its value attribute.       | Yes             | Option value string                                                        | No                          |
| `scroll`          | Scrolls the page (`top`, `bottom`) or an element into view.                 | If value is not `top`/`bottom` | `top`, `bottom`, or empty (uses selector)                              | No                          |
| `screenshot`      | Captures a full-page screenshot. Result attached to task result.            | No              | Optional JPEG quality (0-100, default 90)                                | `base64` (string) or `png` (bytes) |
| `get_dom`         | Retrieves DOM content. Result attached to task result.                      | Optional (defaults to `body`) | No                                                                         | `full_html`, `simplified_html`, `text_content` |
| `run_script`      | Executes arbitrary JavaScript in the page context. Result attached.         | No              | JavaScript code string                                                     | No                          |

## Contribution

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
