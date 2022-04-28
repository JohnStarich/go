package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/johnstarich/go/diffcover/internal/span"
)

var inGitHubActions = os.Getenv("GITHUB_ACTIONS") == "true"

func workflowCommand(command, message string, args map[string]string) string {
	var sb strings.Builder
	sb.WriteString("::")
	sb.WriteString(command)
	sb.WriteRune(' ')
	var argsSlice []string
	for key, value := range args {
		key = githubActionsEncode(key)
		value = githubActionsEncode(value)
		argsSlice = append(argsSlice, fmt.Sprintf("%s=%s", key, value))
	}
	sort.Strings(argsSlice)

	for i, arg := range argsSlice {
		if i != 0 {
			sb.WriteRune(',')
		}
		sb.WriteString(arg)
	}
	sb.WriteString("::")
	message = githubActionsEncode(message)
	sb.WriteString(message)
	return sb.String()
}

// githubActionsEncode encodes a string for workflow command parameters and values.
// Mapping guide found here: https://pakstech.com/blog/github-actions-workflow-commands/#what-are-workflow-commands
func githubActionsEncode(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch r {
		case '%':
			sb.WriteString("%25")
		case '\r':
			sb.WriteString("%0D")
		case '\n':
			sb.WriteString("%0A")
		case ':':
			sb.WriteString("%3A")
		case ',':
			sb.WriteString("%2C")
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func runWorkflow(s string) {
	if inGitHubActions {
		fmt.Println(s)
	}
}

func coverageCommand(percent float64, file string, uncovered []span.Span) string {
	status := newCoverageStatus(percent)
	message := fmt.Sprintf("Diff coverage is %.1f%%", 100*percent)
	args := map[string]string{
		"title": "diffcover",
	}
	if file != "" {
		args["file"] = file
	}
	if len(uncovered) > 0 {
		first := uncovered[0]
		args["title"] = fmt.Sprintf("Not enough tests on %s. (-%.1f%%)", file, 100*(1-percent))
		args["line"] = fmt.Sprintf("%d", first.Start)
		args["endLine"] = fmt.Sprintf("%d", first.End-1)
		message = ""
		for _, lines := range uncovered {
			message += fmt.Sprintf("* %s#L%d-%d\n", file, lines.Start, lines.End-1)
		}
	}
	return workflowCommand(status.WorkflowCommand(), message, args)
}
