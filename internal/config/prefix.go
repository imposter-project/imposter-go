package config

import "path/filepath"

// prefixResponseFiles resolves response file and dir paths relative to the
// config file's directory.
func prefixResponseFiles(resp *Response, relDir string) {
	if resp == nil || relDir == "." {
		return
	}
	if resp.File != "" {
		resp.File = filepath.Join(relDir, resp.File)
	}
	if resp.Dir != "" {
		resp.Dir = filepath.Join(relDir, resp.Dir)
	}
}

// prefixStepFiles resolves step script file paths relative to the config
// file's directory.
func prefixStepFiles(steps []Step, relDir string) {
	if relDir == "." {
		return
	}
	for i := range steps {
		if steps[i].File != "" {
			steps[i].File = filepath.Join(relDir, steps[i].File)
		}
	}
}

// prefixScheduleFiles resolves file paths referenced by schedule entries
// (responses and step scripts) relative to the config file's directory.
func prefixScheduleFiles(schedules []Schedule, relDir string) {
	for i := range schedules {
		prefixResponseFiles(schedules[i].Response, relDir)
		for j := range schedules[i].Responses {
			prefixResponseFiles(&schedules[i].Responses[j], relDir)
		}
		prefixStepFiles(schedules[i].Steps, relDir)
	}
}
