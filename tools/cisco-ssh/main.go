// cisco-ssh is a CLI tool for SSH command execution and file transfer to
// Cisco NX-OS switches. It uses Go's x/crypto/ssh library to handle both
// password and keyboard-interactive authentication, which standard SSH
// ASKPASS mechanisms fail to do on NX-OS.
//
// Usage:
//
//	cisco-ssh [options] <command>
//	cisco-ssh [options] upload <local-file> <remote-path>
//
// Environment variables (used as defaults when flags are not provided):
//
//	SSH_HOST  — switch IP or hostname
//	SSH_USER  — SSH username
//	SSH_PASS  — SSH password
//	SSH_PORT  — SSH port (default: 22)
package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

func main() {
	host := envOrDefault("SSH_HOST", "")
	user := envOrDefault("SSH_USER", "")
	pass := envOrDefault("SSH_PASS", "")
	port := envOrDefault("SSH_PORT", "22")

	// Parse flags from the beginning of os.Args
	args := os.Args[1:]
	for len(args) > 0 {
		switch {
		case args[0] == "--host" && len(args) > 1:
			host = args[1]
			args = args[2:]
		case args[0] == "--user" && len(args) > 1:
			user = args[1]
			args = args[2:]
		case args[0] == "--pass" && len(args) > 1:
			pass = args[1]
			args = args[2:]
		case args[0] == "--port" && len(args) > 1:
			port = args[1]
			args = args[2:]
		case strings.HasPrefix(args[0], "--"):
			fatal("unknown flag: %s", args[0])
		default:
			goto done
		}
	}
done:

	if host == "" || user == "" || pass == "" {
		fatal("SSH_HOST, SSH_USER, SSH_PASS required (env vars or --host/--user/--pass flags)")
	}
	if len(args) == 0 {
		fatal("Usage: cisco-ssh [--host H --user U --pass P] <command>\n       cisco-ssh [--host H --user U --pass P] upload <local> <remote>")
	}

	addr := host + ":" + port
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(pass),
			ssh.KeyboardInteractive(func(_, _ string, questions []string, _ []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range questions {
					answers[i] = pass
				}
				return answers, nil
			}),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		fatal("SSH dial %s: %v", addr, err)
	}
	defer client.Close()

	if args[0] == "upload" {
		if len(args) < 3 {
			fatal("Usage: cisco-ssh upload <local-file> <remote-path>")
		}
		doUpload(client, args[1], args[2])
	} else {
		cmd := strings.Join(args, " ")
		doCommand(client, cmd)
	}
}

func doCommand(client *ssh.Client, cmd string) {
	session, err := client.NewSession()
	if err != nil {
		fatal("Session: %v", err)
	}
	defer session.Close()

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	if err := session.Run(cmd); err != nil {
		// Print the error but use the exit code from SSH if available
		if exitErr, ok := err.(*ssh.ExitError); ok {
			os.Exit(exitErr.ExitStatus())
		}
		fmt.Fprintf(os.Stderr, "Run: %v\n", err)
		os.Exit(1)
	}
}

func doUpload(client *ssh.Client, localPath, remotePath string) {
	f, err := os.Open(localPath)
	if err != nil {
		fatal("Open %s: %v", localPath, err)
	}
	defer f.Close()
	fi, _ := f.Stat()

	session, err := client.NewSession()
	if err != nil {
		fatal("Session: %v", err)
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		fatal("Stdin: %v", err)
	}

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	// NX-OS: use "run bash cat > path" to write via stdin (SCP subsystem is disabled)
	cmd := fmt.Sprintf("run bash cat > %s && chmod +x %s", remotePath, remotePath)
	if err := session.Start(cmd); err != nil {
		fatal("Start: %v", err)
	}

	n, err := io.Copy(stdin, f)
	if err != nil {
		fatal("Copy: %v", err)
	}
	stdin.Close()

	if err := session.Wait(); err != nil {
		fatal("Wait: %v", err)
	}
	fmt.Fprintf(os.Stderr, "Uploaded %s (%d/%d bytes) to %s\n", localPath, n, fi.Size(), remotePath)
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

// portInt converts port string to int for validation
func portInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 22
	}
	return n
}
