package browserutil

import (
	"fmt"
	"os/exec"
	"runtime"
)

func Open(url string) error {
	var cmd string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "linux", "freebsd", "openbsd", "netbsd":
		cmd = "xdg-open"
	default:
		// Only Mac + Linux support for now
		fmt.Println("Please visit:", url)
		return nil
	}

	return exec.Command(cmd, url).Start()
}
