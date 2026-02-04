package main

import (
	"github.com/sqlc-dev/plugin-sdk-go/codegen"

	python "github.com/asavoy/alt-sqlc-gen-python/internal"
)

func main() {
	codegen.Run(python.Generate)
}
