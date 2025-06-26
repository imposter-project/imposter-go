package main

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"github.com/imposter-project/imposter-go/external/handler"
	"os"
	path2 "path"
	"strings"
	"text/template"
)

//go:embed www/*
var www embed.FS

var specPrefixPath string

// serveStaticContent serves static content from the embedded filesystem.
func serveStaticContent(path string) handler.HandlerResponse {
	path = strings.TrimPrefix(path, specPrefixPath)
	if len(path) == 0 {
		return handler.HandlerResponse{StatusCode: 404, Body: []byte("File Not Found")}
	}

	respHeaders := make(map[string]string)

	// index is a special case
	if path == "/" {
		// TODO cache this
		t, err := template.ParseFS(www, "www/index.html.tmpl")
		if err != nil {
			return handler.HandlerResponse{
				StatusCode: 500,
				Body:       []byte(fmt.Sprintf("error parsing template: %v", err.Error())),
			}
		}

		var output bytes.Buffer
		err = t.Execute(&output, specConfigJSON)
		if err != nil {
			return handler.HandlerResponse{
				StatusCode: 500,
				Body:       []byte(fmt.Sprintf("error executing template: %v", err.Error())),
			}
		}
		return handler.HandlerResponse{
			StatusCode: 200,
			Headers:    respHeaders,
			Body:       output.Bytes(),

			// used for MIME type detection
			FileName: "index.html",
		}
	}

	file, err := www.ReadFile("www" + path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return handler.HandlerResponse{StatusCode: 404, Body: []byte("File Not Found")}
		}
		return handler.HandlerResponse{
			StatusCode: 500,
			Body:       []byte(fmt.Sprintf("error reading file: %s - %v", path, err.Error())),
		}
	}
	fileName := path2.Base(path)
	return handler.HandlerResponse{
		StatusCode: 200,
		Headers:    respHeaders,
		Body:       file,

		// used for MIME type detection
		FileName: fileName,
	}
}
