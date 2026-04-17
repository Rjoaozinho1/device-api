FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main cmd/api/main.go

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/main .

EXPOSE 8080

CMD ["./main"]
