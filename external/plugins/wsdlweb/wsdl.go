package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/imposter-project/imposter-go/external/shared"
)

var wsdlConfigs []WSDLConfig

var mu sync.RWMutex
var cachedWSDLs = make(map[string][]byte)

type WSDLConfig struct {
	Name         string `json:"name"`
	URL          string `json:"url"`
	OriginalPath string `json:"-"`
	ConfigDir    string `json:"-"`
}

func generateWSDLConfig(configs []shared.LightweightConfig) error {
	for _, cfg := range configs {
		if cfg.WSDLFile == "" {
			continue
		}
		wsdlFile := strings.TrimPrefix(cfg.WSDLFile, "/")
		wsdlConfigs = append(wsdlConfigs, WSDLConfig{
			Name:         wsdlFile,
			URL:          wsdlPrefixPath + "/wsdl/" + wsdlFile,
			OriginalPath: cfg.WSDLFile,
			ConfigDir:    cfg.ConfigDir,
		})
	}
	return nil
}

// serveRawWSDL serves the WSDL file content.
// If no matching WSDL is found, it returns nil.
func serveRawWSDL(path string) *shared.HandlerResponse {
	for _, wsdlConfig := range wsdlConfigs {
		if path == wsdlConfig.URL {
			return getWSDL(wsdlConfig)
		}
	}
	return nil
}

func getWSDL(wsdlConfig WSDLConfig) *shared.HandlerResponse {
	wsdlPath := filepath.Join(wsdlConfig.ConfigDir, wsdlConfig.OriginalPath)
	data := readWSDLFromCache(wsdlPath)

	if data == nil {
		mu.Lock()
		defer mu.Unlock()

		logger.Trace("loading WSDL file", "path", wsdlPath)
		var err error
		data, err = os.ReadFile(wsdlPath)
		if err != nil {
			if os.IsNotExist(err) {
				return &shared.HandlerResponse{
					StatusCode: 404,
					Body:       []byte("WSDL file not found"),
				}
			}
			return &shared.HandlerResponse{
				StatusCode: 500,
				Body:       []byte(fmt.Sprintf("Error reading WSDL file: %v", err)),
			}
		}

		cachedWSDLs[wsdlPath] = data
	}

	return &shared.HandlerResponse{
		StatusCode: 200,
		Body:       data,
		Headers: map[string]string{
			"Content-Type": "application/xml",
		},
	}
}

func readWSDLFromCache(wsdlPath string) []byte {
	mu.RLock()
	defer mu.RUnlock()
	return cachedWSDLs[wsdlPath]
}
