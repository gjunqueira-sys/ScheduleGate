FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/license-server ./cmd/license-server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /out/license-server /usr/local/bin/license-server
ENV SG_PORT=8080
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/license-server"]
