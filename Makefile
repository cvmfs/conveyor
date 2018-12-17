all: conveyor

conveyor:
	CGO_ENABLED=0 go build

clean:
	go clean
