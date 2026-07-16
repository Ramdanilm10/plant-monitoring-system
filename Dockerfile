# syntax=docker/dockerfile:1

# =========================================================
# Tahap 1: Build frontend React/Vite
# =========================================================
FROM node:24-alpine AS frontend-builder

WORKDIR /app/frontend

COPY frontend/package.json frontend/package-lock.json ./

RUN npm ci

COPY frontend/ ./

ENV VITE_API_BASE_URL=""

RUN npm run build


# =========================================================
# Tahap 2: Build backend Go
# =========================================================
FROM golang:1.26.5-alpine3.24 AS backend-builder

WORKDIR /app/backend

RUN apk add --no-cache ca-certificates

COPY backend/go.mod backend/go.sum ./

RUN go mod download

COPY backend/ ./

RUN CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/plant-monitoring-backend \
    .


# =========================================================
# Tahap 3: Production image
# =========================================================
FROM alpine:3.24

WORKDIR /app/backend

RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    && addgroup -S appgroup \
    && adduser -S appuser -G appgroup

COPY --from=backend-builder \
    /out/plant-monitoring-backend \
    ./plant-monitoring-backend

COPY --from=frontend-builder \
    /app/frontend/dist \
    /app/frontend/dist

ENV APP_ENV="production"

ENV FRONTEND_DIST_PATH="/app/frontend/dist"

ENV TZ="Asia/Jakarta"

EXPOSE 8080

USER appuser

CMD ["./plant-monitoring-backend"]