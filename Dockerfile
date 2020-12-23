
# docker build .
# docker tag b0f9c5da8d7764d70870e25e0f4c98e758ed5aeead0ad5deaca42819a70d067d mrwetsnow/pepper-poker:0.0.1
# docker docker push mrwetsnow/pepper-poker:0.0.1

# Use the offical golang image to create a binary.
# This is based on Debian and sets the GOPATH to /go.
# https://hub.docker.com/_/golang
FROM golang:1.15-buster as builder

# Create and change to the app directory.
WORKDIR /app

# Retrieve application dependencies.
# This allows the container build to reuse cached dependencies.
# Expecting to copy go.mod and if present go.sum.
COPY go.* ./
RUN go mod download

RUN GRPC_HEALTH_PROBE_VERSION=v0.2.2 && \
    wget -q -O /bin/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64 && \
    chmod +x /bin/grpc_health_probe

# Copy local code to the container image.
COPY . ./
COPY ./key/server.key ./key/server.key
COPY ./cert/server.crt ./cert/server.crt

# Build the binary.
# RUN go build -mod=readonly -v -o server
RUN go build -v -o server.bin server/cmd/run.go

# Use the official Debian slim image for a lean production container.
# https://hub.docker.com/_/debian
# https://docs.docker.com/develop/develop-images/multistage-build/#use-multi-stage-builds
FROM debian:buster-slim
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates inetutils-ping telnet netcat && \
    rm -rf /var/lib/apt/lists/*

# Copy the binary to the production image from the builder stage.
COPY --from=builder /app/server.bin /app/server.bin
COPY --from=builder /app/key/server.key /etc/key/server.key
COPY --from=builder /app/cert/server.crt /etc/cert/server.crt
COPY --from=builder /app/server/templates/ /app/server/templates/
COPY --from=builder /app/server/static/ /app/server/static/
# Add grpc-health-probe to use with readiness and liveness probes
COPY --from=builder /bin/grpc_health_probe /bin/grpc_health_probe

RUN chmod +x /app/server.bin
# Run the web service on container startup.
# CMD ["/app/server"]
# ENTRYPOINT ["/app/server", "--enable_datadog"]
