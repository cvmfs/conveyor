all: conveyor

conveyor:
	go build

test:
	go test -v ./...

clean:
	go clean
