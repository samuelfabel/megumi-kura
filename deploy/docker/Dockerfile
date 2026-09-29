# Build frontend
FROM node:22-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json* ./
RUN npm install
COPY web/ ./
RUN npm run build

# Build API
FROM golang:1.22-alpine AS api
WORKDIR /src
COPY api/go.mod api/go.sum ./
RUN go mod download
COPY api/ ./
RUN CGO_ENABLED=0 go build -o /megumi-kura ./cmd/megumi-kura

# Runtime: single binary + static assets + migrations
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=api /megumi-kura /app/megumi-kura
COPY --from=api /src/migrations /app/migrations
COPY --from=web /web/dist /app/web/dist
ENV MK_HTTP_ADDR=:8080
ENV MK_STATIC_DIR=/app/web/dist
EXPOSE 8080
ENTRYPOINT ["/app/megumi-kura", "-migrations", "/app/migrations/postgres"]
