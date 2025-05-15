FROM golang:1.23.1 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod tidy && go mod vendor

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o identity ./cmd/main.go

# ---------- Stage 2: Final ----------
FROM alpine:latest

WORKDIR /app

# Copy built binary
COPY --from=builder /app/identity .


EXPOSE 8081

CMD ["./identity"]