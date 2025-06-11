//go:build ignore
// +build ignore

package main

import (
	"log"
	"net/http"

	"github.com/shurcooL/vfsgen"
)

func main() {

	swaggerFS := http.Dir("../../swagger")

	err := vfsgen.Generate(swaggerFS, vfsgen.Options{
		PackageName:  "http",
		BuildTags:    "!dev",
		VariableName: "SwaggerAssets",
		Filename:     "swagger_assets.go",
	})
	if err != nil {
		log.Fatalln(err)
	}
}
