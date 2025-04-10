package openapi

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/imposter-project/imposter-go/pkg/logger"
)

// resolveSpecFile resolves the spec file path and downloads a remote spec file if needed
func resolveSpecFile(specFile string, configDir string) (string, error) {
	if isURL(specFile) {
		return downloadSpec(specFile)
	}

	if !filepath.IsAbs(specFile) {
		specFile = filepath.Join(configDir, specFile)
	}

	return specFile, nil
}

func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// downloadSpec downloads a spec file from a URL to a temporary file
func downloadSpec(url string) (string, error) {
	logger.Infof("downloading OpenAPI spec from %s", url)

	tmpFile, err := os.CreateTemp("", "openapi-spec-*.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	tmpFileName := tmpFile.Name()

	resp, err := http.Get(url)
	if err != nil {
		os.Remove(tmpFileName)
		return "", fmt.Errorf("failed to download spec file from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		os.Remove(tmpFileName)
		return "", fmt.Errorf("failed to download spec file from %s: HTTP %d", url, resp.StatusCode)
	}

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		os.Remove(tmpFileName)
		return "", fmt.Errorf("failed to write spec file to temporary file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFileName)
		return "", fmt.Errorf("failed to close temporary file: %w", err)
	}

	logger.Debugf("downloaded OpenAPI spec to temporary file: %s", tmpFileName)
	return tmpFileName, nil
}
