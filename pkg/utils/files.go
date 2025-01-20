package utils

import (
	"errors"
	"fmt"
	"github.com/imposter-project/imposter-go/internal/logger"
	"path/filepath"
	"strings"
)

// ValidatePath validates a file path to ensure it is within the config directory
func ValidatePath(path string, configDir string) (string, error) {
	filePath := filepath.Join(configDir, path)
	filePath = filepath.Clean(filePath)

	if !strings.HasPrefix(filePath, filepath.Clean(configDir)+string(filepath.Separator)) {
		msg := fmt.Sprintf("file path escapes config directory: %s", filePath)
		logger.Errorf(msg)
		return "", errors.New(msg)
	}
	return filePath, nil
}
