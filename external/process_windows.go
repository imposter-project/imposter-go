//go:build windows

package external

import "os/exec"

// setPluginProcessAttr is a no-op on Windows; process group isolation is not
// required because Windows does not propagate Unix signals.
func setPluginProcessAttr(_ *exec.Cmd) {}
