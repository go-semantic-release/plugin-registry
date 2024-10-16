FROM golang:1.23 AS builder

ARG VERSION=dev

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY ./ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-extldflags '-static' -s -w -X main.version=${VERSION}" ./cmd/plugin-registry/

FROM gcr.io/distroless/static

COPY --from=builder /app/plugin-registry /

ENV PORT=8080
EXPOSE 8080

ENTRYPOINT ["/plugin-registry"]
