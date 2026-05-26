.PHONY: tidy build run clean

# CGO_ENABLED=0 disables Cgo to produce a statically-linked Go binary, making the executable more portable and container-friendly.
# A statically-linked binary doesn't have run-time dependencies on other libraries, not even the C runtime library (libc)
build:
	CGO_ENABLED=0 go build -o ./bin/waitingroom-admissionservice .

tidy:
	go mod tidy

run:
	go run .

clean:
	rm -f ./bin/waitingroom-admissionservice
