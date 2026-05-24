.PHONY: tidy build run clean

build:
	go build -o ./bin/waitingroom-admissionservice .

tidy:
	go mod tidy

run:
	go run .

clean:
	rm -f ./bin/waitingroom-admissionservice
