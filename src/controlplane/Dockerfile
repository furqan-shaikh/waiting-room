# Stage 1: Build go binary
FROM golang:1.26.2 AS builder
WORKDIR /app
COPY . .
# RUN make file
WORKDIR /app/controlplane
RUN make build

# Stage 2: Final production stage
FROM alpine:3.23
WORKDIR /control-plane
COPY --from=builder /app/controlplane/bin/waitingroom-controlplane .
EXPOSE 3000
# Runtime image does not have Go source or Go toolchain. start command should run the compiled binary directly.
CMD ["./waitingroom-controlplane"]
