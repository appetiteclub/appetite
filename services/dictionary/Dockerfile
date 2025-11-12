FROM golang:1.24-alpine AS builder
RUN apk add --no-cache gcc musl-dev

WORKDIR /build

# Copy workspace files needed for the build
COPY go.work go.work.sum ./
COPY pkg/ pkg/
COPY services/ services/

WORKDIR /build/services/dictionary

RUN go mod download
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o fake .

FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /app

COPY --from=builder /build/services/dictionary/dictionary .
COPY --from=builder /build/services/dictionary/config.yaml .

RUN mkdir -p /data

EXPOSE 8085

CMD ["./dictionary"]
