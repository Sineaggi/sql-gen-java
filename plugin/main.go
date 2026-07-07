package main

import (
	"github.com/sqlc-dev/plugin-sdk-go/codegen"

	javagen "github.com/sqlc-dev/sqlc-gen-java/internal"
)

func main() {
	codegen.Run(javagen.Generate)
}
