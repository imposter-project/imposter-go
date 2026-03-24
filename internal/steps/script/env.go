package script

import "os"

// buildEnv returns a map of environment variables for use in scripts.
func buildEnv() map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		for i := 0; i < len(e); i++ {
			if e[i] == '=' {
				env[e[:i]] = e[i+1:]
				break
			}
		}
	}
	return env
}

// buildConfig returns a config object exposing configuration properties to scripts.
func buildConfig(configDir string) map[string]string {
	return map[string]string{
		"dir": configDir,
	}
}
