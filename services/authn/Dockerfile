FROM golang:1.24-alpine AS builder
RUN apk add --no-cache gcc musl-dev

WORKDIR /build

# Copy workspace files
COPY go.work go.work.sum ./
COPY pkg/ pkg/
COPY services/ services/

WORKDIR /build/services/authn

RUN go mod download
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o authn .

FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /app

COPY --from=builder /build/services/authn/authn .
COPY --from=builder /build/services/authn/config.yaml .

RUN mkdir -p /data

EXPOSE 8082

CMD ["./authn"]
