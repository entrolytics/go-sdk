.PHONY: lint lint-fix format test

lint:
	golangci-lint run ./...

lint-fix:
	golangci-lint run --fix ./...

format:
	gofmt -w .
	goimports -w .

test:
	go test -v ./...
