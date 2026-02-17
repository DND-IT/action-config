// Package outputs provides GitHub Actions output and logging utilities.
package outputs

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// SetOutput writes a value to GITHUB_OUTPUT.
func SetOutput(name, value string) {
	outputFile := os.Getenv("GITHUB_OUTPUT")
	if outputFile == "" {
		fmt.Printf("::set-output name=%s::%s\n", name, value)
		return
	}

	// Use os.Stdout directly to avoid a second file descriptor that
	// races with fmt.Print* and causes truncated output.
	var f *os.File
	if outputFile == "/dev/stdout" {
		f = os.Stdout
	} else if outputFile == "/dev/stderr" {
		f = os.Stderr
	} else {
		var err error
		f, err = os.OpenFile(outputFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			fmt.Printf("::set-output name=%s::%s\n", name, value)
			return
		}
		defer func() { _ = f.Close() }()
	}

	if strings.Contains(value, "\n") {
		delimiter := fmt.Sprintf("ghadelimiter_%d", time.Now().UnixNano())
		_, _ = fmt.Fprintf(f, "%s<<%s\n%s\n%s\n", name, delimiter, value, delimiter)
	} else {
		_, _ = fmt.Fprintf(f, "%s=%s\n", name, value)
	}
}

// LogInfo prints an info message.
func LogInfo(msg string) {
	fmt.Println(msg)
}

// LogNotice prints a notice message.
func LogNotice(msg string) {
	fmt.Printf("::notice::%s\n", msg)
}

// LogError prints an error message.
func LogError(msg string) {
	fmt.Printf("::error::%s\n", msg)
}
