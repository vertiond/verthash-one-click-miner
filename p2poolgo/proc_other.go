//go:build !windows

package p2poolgo

import "os/exec"

// prepareNodeCommand sets OS-specific process attributes. No Pdeathsig on
// Linux: it fires when the forking OS thread exits, which Go may retire while
// the node is healthy. The node polls -parentpid instead.
func prepareNodeCommand(cmd *exec.Cmd) {}

func afterNodeStart(cmd *exec.Cmd) error { return nil }
