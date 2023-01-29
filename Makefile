build:
	go build -ldflags "-s -w"

run:
	go run main.go run

protolint:
	buf lint

protogen:
	buf generate
