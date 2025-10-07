# build stage
FROM golang:1.22 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./main.go

# run stage
FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=builder /app/server /app/server
COPY config.docker.yaml /app/config.yaml
COPY docs/swagger.yaml /app/docs/swagger.yaml

# env at runtime:
# - JWT_SECRET, MYSQL_DSN, REDIS_ADDR, REDIS_PASSWORD
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/app/server"]
