FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod .
RUN go mod tidy
COPY . .
RUN GOOS=linux GOARCH=amd64 go build -o gateway main.go

FROM alpine:3.14
WORKDIR /app
COPY --from=builder /app/gateway .
EXPOSE 6969
CMD ["./gateway"]
