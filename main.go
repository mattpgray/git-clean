// Short script for performing cleanup on a git repo. Use the
// flag -h for options. Performs a dry run by default. Provide the
// flag -f to perform the commands.
package main

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var (
	force   = flag.Bool("f", false, "Perform cleanup options.")
	verbose = flag.Bool("v", false, "Verbose output.")
)

func main() {
	flag.Parse()

	currentBranch := getCurrentBranch()
	{
		defaultBranch := getDefaultBranch()
		if currentBranch != defaultBranch {
			fmt.Fprintf(os.Stderr, "[error] Refusing to run `git-clean` without being on the default branch. Currently on branch %s. Default branch %s.\n", currentBranch, defaultBranch)
			os.Exit(1)
		}
	}

	mergedBranches := getMergedBranches()
	for _, branch := range mergedBranches {
		msg := "[deleting]"
		if !*force {
			msg = "[would delete]"
		}
		fmt.Printf("%s %s\n", msg, branch)
		if *force {
			deleteBranch(branch)
		}
	}
}

var defaultBranchRegexp = regexp.MustCompile(`HEAD branch: (.*)`)

func getDefaultBranch() string {
	out := runCmdDefaultTimeout("git", "remote", "show", "origin")
	bb := defaultBranchRegexp.FindSubmatch(out)
	if len(bb) != 2 {
		fmt.Fprint(os.Stderr, "[error] failed to extract default branch")
		os.Exit(1)
	}

	return strings.TrimSpace(string(bb[1]))
}

func getCurrentBranch() string {
	out := runCmdDefaultTimeout("git", "rev-parse", "--abbrev-ref", "HEAD")
	return string(bytes.TrimSpace(out))
}

func getMergedBranches() []string {
	out := runCmdDefaultTimeout("git", "branch", "--merged")
	scanner := bufio.NewScanner(bytes.NewReader(out))
	branches := []string{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "* ") {
			branches = append(branches, line)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "[error] scanner error %v\n", err)
	}
	return branches
}

func deleteBranch(branch string) {
	runCmdDefaultTimeout("git", "branch", "-d", branch)
}

func runCmdDefaultTimeout(name string, args ...string) []byte {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	return runCmd(ctx, name, args...)
}

func runCmd(ctx context.Context, name string, args ...string) []byte {
	cmd := exec.CommandContext(ctx, name, args...)
	// FIXME: use a command strings that can be copy-pasted into the shell
	if *verbose {
		fmt.Printf("[cmd] %s\n", cmd.String())
	} // if

	cmd.Stderr = &prefixWriter{prefix: "[cmd][stderr] ", w: os.Stderr}
	stdout := &bytes.Buffer{}
	if *verbose {
		cmd.Stdout = io.MultiWriter(&prefixWriter{prefix: "[cmd][stdout] ", w: os.Stdout}, stdout)
	} else {
		cmd.Stdout = stdout
	}

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "[error] error running command %s, %v\n", cmd.String(), err)
		os.Exit(1)
	}

	return stdout.Bytes()
}

type prefixWriter struct {
	started bool
	prefix  string
	w       io.Writer
}

func (pw *prefixWriter) Write(p []byte) (int, error) {
	for start := 0; start < len(p); {
		newlineIdx := -1
		for i := start; i < len(p); i++ {
			if p[i] == '\n' {
				newlineIdx = i
				break
			}
		}
		if newlineIdx == -1 {
			// in a line
			n, err := pw.writeOnce(p[start:])
			if err == nil {
				pw.started = true
			}
			return start + n, err
		}
		n, err := pw.writeOnce(p[start : newlineIdx+1])
		if err != nil {
			return start + n, err
		}
		pw.started = false
		start = newlineIdx + 1
	}
	return len(p), nil
}

func (pw *prefixWriter) writeOnce(p []byte) (int, error) {
	if !pw.started {
		if _, err := pw.w.Write([]byte(pw.prefix)); err != nil {
			return 0, err
		}
	}
	return pw.w.Write(p)
}
