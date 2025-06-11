//go:build dev
// +build dev

package http

import "net/http"

var SwaggerAssets http.FileSystem = http.Dir("../../swagger")
