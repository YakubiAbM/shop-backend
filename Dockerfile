# ЭТАП 1: КОМПИЛЯЦИЯ
FROM golang:1.23 AS builder
WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

# ЭТАП 2: ЗАПУСК (Используем легкий образ)
FROM gcr.io/distroless/static-debian11
WORKDIR /
COPY --from=builder /app/server /server
EXPOSE 8080
CMD ["/server"]