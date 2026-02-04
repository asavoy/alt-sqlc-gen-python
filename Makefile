.PHONY: build test

build:
	go build ./...

test: bin/alt-sqlc-gen-python.wasm
	go test ./...

all: bin/alt-sqlc-gen-python bin/alt-sqlc-gen-python.wasm

bin/alt-sqlc-gen-python: bin go.mod go.sum $(wildcard **/*.go)
	cd plugin && go build -o ../bin/alt-sqlc-gen-python ./main.go

bin/alt-sqlc-gen-python.wasm: bin/alt-sqlc-gen-python
	cd plugin && GOOS=wasip1 GOARCH=wasm go build -o ../bin/alt-sqlc-gen-python.wasm main.go
	shasum -a 256 bin/alt-sqlc-gen-python.wasm

bin:
	mkdir -p bin
