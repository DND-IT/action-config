// Package outputs provides GitHub Actions output and logging utilities.
package outputs

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type outputEntry struct {
	name  string
	value string
}

var recorded []outputEntry

// SetOutput writes a value to GITHUB_OUTPUT.
func SetOutput(name, value string) {
	recorded = append(recorded, outputEntry{name, value})
	fmt.Printf("::debug::output %s=%s\n", name, value)
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

// WriteSummary writes all recorded outputs to the GitHub Actions step summary.
func WriteSummary() {
	summaryFile := os.Getenv("GITHUB_STEP_SUMMARY")
	if summaryFile == "" || len(recorded) == 0 {
		return
	}

	var sb strings.Builder
	sb.WriteString("### Outputs\n\n")
	sb.WriteString("| Name | Value |\n")
	sb.WriteString("|------|-------|\n")

	for _, e := range recorded {
		if strings.Contains(e.value, "\n") || len(e.value) > 100 {
			sb.WriteString(fmt.Sprintf("| `%s` | <details><summary>Show</summary>\n\n```json\n%s\n```\n\n</details> |\n", e.name, e.value))
		} else {
			sb.WriteString(fmt.Sprintf("| `%s` | `%s` |\n", e.name, e.value))
		}
	}

	f, err := os.OpenFile(summaryFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	_, _ = f.WriteString(sb.String())
}
