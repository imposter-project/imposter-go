package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/imposter-project/imposter-go/pkg/logger"

	"gopkg.in/yaml.v3"
)

// LoadImposterConfig loads configurations from environment variables
func LoadImposterConfig() *ImposterConfig {
	port := os.Getenv("IMPOSTER_PORT")
	if port == "" {
		port = "8080" // Default port
	}

	serverUrl := os.Getenv("IMPOSTER_SERVER_URL")
	if serverUrl == "" {
		var hostSuffix string
		if port != "80" {
			hostSuffix = fmt.Sprintf(":%s", port)
		}
		serverUrl = fmt.Sprintf("http://localhost%s", hostSuffix)
	}

	return &ImposterConfig{
		LegacyConfigSupported: isLegacyConfigEnabled(),
		ServerPort:            port,
		ServerUrl:             serverUrl,
	}
}

// LoadConfig loads all config files in the specified directory
func LoadConfig(configDir string, imposterConfig *ImposterConfig) []Config {
	logger.Debugf("loading config files from %s", configDir)
	var configs []Config

	scanRecursive := (os.Getenv("IMPOSTER_CONFIG_SCAN_RECURSIVE") == "true")
	autoBasePath := (os.Getenv("IMPOSTER_AUTO_BASE_PATH") == "true")

	ignorePaths := loadIgnorePaths(configDir)

	err := filepath.Walk(configDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip ignored paths
		if shouldIgnorePath(path, ignorePaths) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip subdirectories if not scanning recursively
		if info.IsDir() && info.Name() != filepath.Base(configDir) && !scanRecursive {
			return filepath.SkipDir
		}

		if !info.IsDir() && (strings.HasSuffix(info.Name(), "-config.json") || strings.HasSuffix(info.Name(), "-config.yaml") || strings.HasSuffix(info.Name(), "-config.yml")) {
			logger.Infof("loading config file: %s", path)
			fileConfig, err := parseConfig(path, imposterConfig)
			if err != nil {
				return err
			}

			// Set basePath if autoBasePath is enabled
			if autoBasePath && fileConfig.BasePath == "" {
				baseDir := filepath.Dir(path)
				relDir, err := filepath.Rel(configDir, baseDir)
				if err != nil {
					return err
				}
				if relDir != "." {
					fileConfig.BasePath = "/" + relDir
				}
			}

			// Prefix referenced files with relative directory if in a subdirectory
			baseDir := filepath.Dir(path)
			relDir, err := filepath.Rel(configDir, baseDir)
			if err != nil {
				return err
			}

			for i := range fileConfig.Resources {
				// Resolve response file path relative to config file
				if fileConfig.Resources[i].Response.File != "" && relDir != "." {
					fileConfig.Resources[i].Response.File = filepath.Join(relDir, fileConfig.Resources[i].Response.File)
				}
				// Resolve response dir path relative to config file
				if fileConfig.Resources[i].Response.Dir != "" && relDir != "." {
					fileConfig.Resources[i].Response.Dir = filepath.Join(relDir, fileConfig.Resources[i].Response.Dir)
				}
				// Prefix paths with basePath
				if fileConfig.BasePath != "" {
					fileConfig.Resources[i].Path = filepath.Join(fileConfig.BasePath, fileConfig.Resources[i].Path)
				}

				// Prefix step script files with relative directory
				if fileConfig.Resources[i].Steps != nil {
					for j := range fileConfig.Resources[i].Steps {
						if fileConfig.Resources[i].Steps[j].File != "" {
							fileConfig.Resources[i].Steps[j].File = filepath.Join(relDir, fileConfig.Resources[i].Steps[j].File)
						}
					}
				}
			}

			if fileConfig.Plugin == "openapi" {
				// Resolve OpenAPI spec path relative to config file
				if fileConfig.SpecFile != "" && !filepath.IsAbs(fileConfig.SpecFile) {
					fileConfig.SpecFile = filepath.Join(relDir, fileConfig.SpecFile)
				}
			} else if fileConfig.Plugin == "soap" {
				// Resolve WSDL file path relative to config file
				if fileConfig.WSDLFile != "" && !filepath.IsAbs(fileConfig.WSDLFile) {
					fileConfig.WSDLFile = filepath.Join(relDir, fileConfig.WSDLFile)
				}
			}

			if fileConfig.System != nil {
				// Resolve preload file paths relative to config file
				for storeName := range fileConfig.System.Stores {
					store := fileConfig.System.Stores[storeName]
					if store.PreloadFile != "" && !filepath.IsAbs(store.PreloadFile) {
						store.PreloadFile = filepath.Join(relDir, store.PreloadFile)
					}
					fileConfig.System.Stores[storeName] = store
				}
			}

			configs = append(configs, *fileConfig)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	// Validate plugin types
	for _, cfg := range configs {
		switch cfg.Plugin {
		case "openapi", "rest", "soap":
			// Valid plugins
		default:
			panic("Unsupported plugin type: " + cfg.Plugin)
		}
	}

	return configs
}

// loadIgnorePaths loads ignore paths from .imposterignore file or uses default ignore paths
func loadIgnorePaths(configDir string) []string {
	defaultIgnorePaths := []string{".git", ".idea", ".svn", "node_modules"}
	ignoreFilePath := filepath.Join(configDir, ".imposterignore")

	if _, err := os.Stat(ignoreFilePath); os.IsNotExist(err) {
		return defaultIgnorePaths
	}

	data, err := os.ReadFile(ignoreFilePath)
	if err != nil {
		logger.Warnf("failed to read .imposterignore file: %v", err)
		return defaultIgnorePaths
	}

	lines := strings.Split(string(data), "\n")
	var ignorePaths []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			ignorePaths = append(ignorePaths, line)
		}
	}

	return ignorePaths
}

// shouldIgnorePath checks if a path should be ignored based on the ignore paths
func shouldIgnorePath(path string, ignorePaths []string) bool {
	for _, ignorePath := range ignorePaths {
		if strings.Contains(path, ignorePath) {
			return true
		}
	}
	return false
}

// parseConfig loads and parses a YAML configuration file
func parseConfig(path string, imposterConfig *ImposterConfig) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Substitute environment variables
	data = []byte(substituteEnvVars(string(data)))

	var cfg *Config

	// Transform legacy config if legacy support is enabled
	if imposterConfig.LegacyConfigSupported {
		logger.Debugf("legacy config support enabled for %s, attempting transformation...", path)
		cfg, err = transformLegacyConfig(data)
		if err != nil {
			return nil, fmt.Errorf("failed to transform legacy config: %w", err)
		}
	} else {
		// Parse as current format
		cfg = &Config{}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
		}
	}

	// Transform security config into interceptors if present
	transformSecurityConfig(cfg)

	return cfg, nil
}

// substituteEnvVars replaces ${env.VAR} and ${env.VAR:-default} with environment variable values
func substituteEnvVars(content string) string {
	re := regexp.MustCompile(`\$\{env\.([A-Z0-9_]+)(:-([^}]+))?\}`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		groups := re.FindStringSubmatch(match)
		envVar := groups[1]
		defaultValue := groups[3]
		if value, exists := os.LookupEnv(envVar); exists {
			return value
		}
		return defaultValue
	})
}
