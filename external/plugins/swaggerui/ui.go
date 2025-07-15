package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/imposter-project/imposter-go/external/shared"
	"os"
	path2 "path"
	"strings"
	"text/template"
)

//go:embed www/*
var www embed.FS

var specPrefixPath string

// indexResp is the cached response for the index page.
var indexResp shared.HandlerResponse

func init() {
	specPrefixPath = os.Getenv("IMPOSTER_OPENAPI_SPEC_PATH_PREFIX")
	if specPrefixPath == "" {
		specPrefixPath = "/_spec"
	}
}

// serveStaticContent serves static content from the embedded filesystem.
func serveStaticContent(path string) shared.HandlerResponse {
	path = strings.TrimPrefix(path, specPrefixPath)
	if len(path) == 0 {
		return shared.HandlerResponse{StatusCode: 302, Headers: map[string]string{
			"Location": specPrefixPath + "/",
		}}
	}

	// index is a special case
	if path == "/" {
		return indexResp
	}

	file, err := www.ReadFile("www" + path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return shared.HandlerResponse{StatusCode: 404, Body: []byte("File Not Found")}
		}
		return shared.HandlerResponse{
			StatusCode: 500,
			Body:       []byte(fmt.Sprintf("error reading file: %s - %v", path, err.Error())),
		}
	}
	fileName := path2.Base(path)
	return shared.HandlerResponse{
		StatusCode: 200,
		Body:       file,

		// used for MIME type detection
		FileName: fileName,
	}
}

// generateIndexPage generates the index page response using the embedded template.
func generateIndexPage() error {
	// serialise the specConfigs to JSON
	jsonData, err := json.Marshal(specConfigs)
	if err != nil {
		return fmt.Errorf("failed to marshal spec config JSON: %w", err)
	}
	specConfigJSON := string(jsonData)

	// populate the template with the specConfigJSON
	t, err := template.ParseFS(www, "www/index.html.tmpl")
	if err != nil {
		return fmt.Errorf("error parsing template: %v", err.Error())
	}
	var output bytes.Buffer
	err = t.Execute(&output, specConfigJSON)
	if err != nil {
		return fmt.Errorf("error executing template: %v", err.Error())
	}

	// cache the index response
	indexResp = shared.HandlerResponse{
		StatusCode: 200,
		Body:       output.Bytes(),

		// used for MIME type detection
		FileName: "index.html",
	}
	return nil
}
