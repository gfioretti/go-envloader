.PHONY: vet test

vet:
	go vet ./...

test:
	go test -v -race ./...