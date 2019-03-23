// +build ignore

package main

import (
	"net/http"

	"github.com/shurcooL/vfsgen"
)

func main() {
	err := vfsgen.Generate(http.Dir("."), vfsgen.Options{
		PackageName:  "public",
		VariableName: "Assets",
		Filename:     "public_assets.go",
	})
	if err != nil {
		panic(err)
	}
}
