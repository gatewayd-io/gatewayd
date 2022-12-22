build:
	go build -ldflags "-s -w"

run:
	go mod tidy && go run main.go run

clean:
	go clean -testcache

test:
	go test -v ./...

protolint:
	buf lint

protogen:
	buf generate
