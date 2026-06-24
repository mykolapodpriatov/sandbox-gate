# Build a static-ish single binary, then ship it on alpine with the docker CLI so
# the container can drive a mounted Docker socket (-v /var/run/docker.sock:...).
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /sandbox-gate ./cmd/sandbox-gate

FROM alpine:3
RUN apk add --no-cache docker-cli
COPY --from=build /sandbox-gate /usr/local/bin/sandbox-gate
ENTRYPOINT ["sandbox-gate"]
