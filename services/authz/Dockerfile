FROM golang:1.24-alpine AS builder
RUN apk add --no-cache gcc musl-dev

WORKDIR /build

# Copy workspace files
COPY go.work go.work.sum ./
COPY pkg/ pkg/
COPY services/ services/

WORKDIR /build/services/authz

RUN go mod download
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o authz .

FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /app

COPY --from=builder /build/services/authz/authz .
COPY --from=builder /build/services/authz/config.yaml .

RUN mkdir -p /data

EXPOSE 8083

CMD ["./authz"]
