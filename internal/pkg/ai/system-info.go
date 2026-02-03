package ai

import "os"

func getShell() string {
	if os.Getenv("ZSH_VERSION") != "" {
		return "zsh"
	} else {
		// TODO: windows?
		return "bash"
	}
}
