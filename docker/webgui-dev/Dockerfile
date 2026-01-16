# Development container for WebGUI with hot-reloading
FROM golang:1.24-alpine

# Install git (for go mod) and wgo file watcher
RUN apk add --no-cache git && \
    go install github.com/bokwoon95/wgo@latest

WORKDIR /app

# wgo will watch for changes and rebuild/restart the server
# Watching: .go (source), .html (templates), .css (styles)
CMD ["wgo", "run", "-file", ".go", "-file", ".html", "-file", ".css", "./cmd/brains", "gui", "--port", "9981"]
