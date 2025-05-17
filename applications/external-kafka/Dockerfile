# ---------- Stage 1: Build ----------
    FROM golang:1.23.1 AS builder

    WORKDIR /app
    
    COPY go.mod go.sum ./
    RUN go mod tidy && go mod vendor
    
    COPY . .
    
    RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o external-kafka ./cmd/main.go
    
    # ---------- Stage 2: Final ----------
    FROM alpine:latest
    
    WORKDIR /app
    COPY --from=builder /app/external-kafka .
    
    EXPOSE 8088
    CMD ["./external-kafka"]
    