# Quickstart: WebGUI Container Development

## Prerequisites

- Docker and Docker Compose installed
- Task CLI installed (`brew install go-task` or see https://taskfile.dev)

## Starting Development Server

```bash
# Start the WebGUI in development mode with hot-reloading
task webgui:dev
```

The WebGUI will be available at: **http://localhost:9981**

## How It Works

1. Docker Compose starts a container with:
   - Go 1.24 + wgo file watcher
   - Source code mounted for live editing
   - SQLite data persisted in `.data/` directory

2. wgo watches for file changes:
   - `.go` files (Go source)
   - `.html` files (templates)
   - `.css` files (styles)

3. On file save:
   - wgo rebuilds and restarts the server
   - Changes visible within ~5 seconds

## Stopping the Server

Press `Ctrl+C` in the terminal running the task, or:

```bash
# In another terminal
docker compose stop webgui-dev
```

## Data Persistence

SQLite database is stored in `.data/` directory on the host. This directory:
- Survives container restarts
- Is gitignored (local development data only)
- Can be deleted to reset all data

## Troubleshooting

### Port 9981 already in use
```bash
# Find and kill the process using the port
lsof -ti:9981 | xargs kill -9
```

### Changes not detected
```bash
# Restart the container
docker compose restart webgui-dev
```

### Fresh start (reset everything)
```bash
# Stop container and remove data
docker compose down
rm -rf .data/
task webgui:dev
```
