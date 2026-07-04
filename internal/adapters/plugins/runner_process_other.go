//go:build !unix

package plugins

import "os/exec"

func configurePluginCommandProcess(*exec.Cmd) {}
