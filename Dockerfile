# Stage 1: build SvelteKit frontend.
FROM node:22-alpine AS frontend-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ .
RUN npm run build

# Stage 2: build Go backend (with embedded frontend).
FROM golang:1.22-alpine AS go-builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Copy the built frontend into the embed directory.
COPY --from=frontend-builder /app/web/build ./internal/api/frontend
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /timbre-server ./cmd/server

# Stage 3: minimal runtime image.
FROM alpine:3.20
RUN apk add --no-cache ffmpeg ca-certificates tzdata
WORKDIR /app
COPY --from=go-builder /timbre-server .

ENV MS_DATA_DIR=/data \
    MS_PORT=8080 \
    MS_HOST=0.0.0.0 \
    MS_DB_DRIVER=sqlite

EXPOSE 8080
VOLUME ["/data"]

ENTRYPOINT ["./timbre-server"]
