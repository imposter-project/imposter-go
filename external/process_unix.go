//go:build !windows

package external

import (
	"os/exec"
	"syscall"
)

// setPluginProcessAttr places the plugin in its own process group so it does
// not receive signals (e.g. SIGTERM) sent to the parent process group.
func setPluginProcessAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
