package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	path2 "path"
	"strings"
	"text/template"

	"github.com/imposter-project/imposter-go/external/shared"
)

//go:embed www/*
var www embed.FS

var wsdlPrefixPath string

// initialiserResp is the cached response for the initialiser script.
var initialiserResp shared.HandlerResponse

func init() {
	wsdlPrefixPath = os.Getenv("IMPOSTER_WSDL_SPEC_PATH_PREFIX")
	if wsdlPrefixPath == "" {
		wsdlPrefixPath = "/_wsdl"
	}
}

// serveStaticContent serves static content from the embedded filesystem.
func serveStaticContent(path string) shared.HandlerResponse {
	path = strings.TrimPrefix(path, wsdlPrefixPath)
	if len(path) == 0 {
		return shared.HandlerResponse{StatusCode: 302, Headers: map[string]string{
			"Location": wsdlPrefixPath + "/",
		}}
	}

	if path == "/" {
		path = "/index.html"
	}

	// initialiser is a special case
	if path == "/wsdl-initializer.js" {
		return initialiserResp
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

// generateInitialiser generates the initialiser script response using the embedded template.
func generateInitialiser() error {
	// serialise the wsdlConfigs to JSON
	jsonData, err := json.Marshal(wsdlConfigs)
	if err != nil {
		return fmt.Errorf("failed to marshal WSDL config JSON: %w", err)
	}
	wsdlConfigJSON := string(jsonData)

	// populate the template with the wsdlConfigJSON
	t, err := template.ParseFS(www, "www/wsdl-initializer.js.tmpl")
	if err != nil {
		return fmt.Errorf("error parsing template: %v", err.Error())
	}
	var output bytes.Buffer
	err = t.Execute(&output, wsdlConfigJSON)
	if err != nil {
		return fmt.Errorf("error executing template: %v", err.Error())
	}

	// cache the initialiser response
	initialiserResp = shared.HandlerResponse{
		StatusCode: 200,
		Body:       output.Bytes(),

		// used for MIME type detection
		FileName: "wsdl-initializer.js",
	}
	return nil
}
