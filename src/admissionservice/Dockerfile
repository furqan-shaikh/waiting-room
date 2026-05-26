# Stage 1: Build go binary
FROM golang:1.26.2 AS builder
WORKDIR /app
COPY . .
# RUN make file
WORKDIR /app/admissionservice
RUN make build

# Stage 2: Final production stage
FROM alpine:3.23
WORKDIR /admissionservice
COPY --from=builder /app/admissionservice/bin/waitingroom-admissionservice .
EXPOSE 3333
# Runtime image does not have Go source or Go toolchain. start command should run the compiled binary directly.
CMD ["./waitingroom-admissionservice"]
