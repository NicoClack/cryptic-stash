ARG APP_PORT=8080

# Stage 1: Build Frontend with official Node Alpine image
FROM node:24-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
ENV PUBLIC_API_DOMAIN=""
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Backend with Alpine
FROM golang:1.25-alpine AS backend-builder
RUN apk add --no-cache git build-base
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./

# Copy frontend build output to backend/server/public
COPY --from=frontend-builder /app/frontend/build ./server/public
RUN rm ./server/public/no-frontend.html

RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Stage 3: Final lightweight image
FROM alpine:latest
# Install ca-certificates for HTTPS and sqlite runtime
RUN apk --no-cache add ca-certificates sqlite-libs

WORKDIR /root/
# Copy the binary from backend-builder
COPY --from=backend-builder /app/backend/main .

ARG APP_PORT
ENV PORT=${APP_PORT}
ENV CORS_ALLOWED_ORIGINS=""
EXPOSE ${APP_PORT}
CMD ["./main"]
