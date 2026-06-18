package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/domain/session"
	"github.com/phin-tech/whisk/internal/protocol"
)

func runSession(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: whisk session <list|create|update|set-root|close|pty>")
	}
	switch args[0] {
	case "list":
		return runSessionList(args[1:])
	case "create":
		return runSessionCreate(args[1:])
	case "update":
		return runSessionUpdate(args[1:])
	case "set-root":
		return runSessionSetRoot(args[1:])
	case "close":
		return runSessionClose(args[1:])
	case "pty":
		return runSessionPTY(args[1:])
	default:
		return fmt.Errorf("usage: whisk session <list|create|update|set-root|close|pty>")
	}
}

func runSessionList(args []string) error {
	flags := flag.NewFlagSet("session list", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	projectID := flags.String("project", "", "filter by project id")
	outputJSON := flags.Bool("json", false, "write JSON output")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: whisk session list [-project project-id] [-json] [-url http://127.0.0.1:8787]")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	sessions, err := client.NewHTTP(*baseURL, nil).ListSessions(ctx)
	if err != nil {
		return err
	}
	if *projectID != "" {
		filtered := sessions[:0]
		for _, candidate := range sessions {
			if candidate.ProjectID == *projectID {
				filtered = append(filtered, candidate)
			}
		}
		sessions = filtered
	}
	if *outputJSON {
		return printJSON(sessions)
	}
	printSessions(sessions)
	return nil
}

func runSessionCreate(args []string) error {
	flags := flag.NewFlagSet("session create", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	name := flags.String("name", "", "session name")
	rootDir := flags.String("root", "", "session root directory")
	workingDir := flags.String("working-dir", "", "initial pane working directory")
	projectID := flags.String("project", envOrDefault("WHISK_PROJECT_ID", envOrDefault("WHISK_PROJECT", "")), "project id")
	command := flags.String("command", "", "initial command to run in the PTY shell")
	startPTY := flags.Bool("pty", true, "start an initial PTY")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *rootDir == "" {
		return fmt.Errorf("usage: whisk session create -root <path> [-working-dir path] [-project project-id] [-name name] [-command command] [-pty=false] [-url http://127.0.0.1:8787]")
	}
	resolvedRoot, err := filepath.Abs(*rootDir)
	if err != nil {
		return err
	}
	resolvedWorkingDir := ""
	if *workingDir != "" {
		resolvedWorkingDir, err = filepath.Abs(*workingDir)
		if err != nil {
			return err
		}
	}

	req := protocol.CreateSessionRequest{
		Name:       *name,
		RootDir:    resolvedRoot,
		WorkingDir: resolvedWorkingDir,
		ProjectID:  *projectID,
	}
	if *startPTY {
		req.InitialPTY = &protocol.StartPTYOptions{Command: *command}
	} else if *command != "" {
		return fmt.Errorf("-command requires an initial PTY")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	created, err := client.NewHTTP(*baseURL, nil).CreateSession(ctx, req)
	if err != nil {
		return err
	}
	fmt.Println(created.Session.ID)
	return nil
}

func runSessionUpdate(args []string) error {
	flags := flag.NewFlagSet("session update", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	projectID := flags.String("project", "", "project id")
	clearProject := flags.Bool("clear-project", false, "clear project assignment")
	outputJSON := flags.Bool("json", false, "write JSON output")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 || (*projectID == "" && !*clearProject) || (*projectID != "" && *clearProject) {
		return fmt.Errorf("usage: whisk session update <session-id> (-project project-id | -clear-project) [-json] [-url http://127.0.0.1:8787]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	updated, err := client.NewHTTP(*baseURL, nil).SetSessionProject(ctx, protocol.SetSessionProjectRequest{
		SessionID: flags.Arg(0),
		ProjectID: *projectID,
	})
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(updated)
	}
	fmt.Println(updated.ID)
	return nil
}

func runSessionSetRoot(args []string) error {
	flags := flag.NewFlagSet("session set-root", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 2 {
		return fmt.Errorf("usage: whisk session set-root [-url http://127.0.0.1:8787] <session-id> <path>")
	}
	resolvedRoot, err := filepath.Abs(flags.Arg(1))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	updated, err := client.NewHTTP(*baseURL, nil).SetSessionRootDir(ctx, protocol.SetSessionRootDirRequest{
		SessionID: flags.Arg(0),
		RootDir:   resolvedRoot,
	})
	if err != nil {
		return err
	}
	fmt.Printf("%s\t%s\n", updated.ID, updated.RootDir)
	return nil
}

func runSessionClose(args []string) error {
	flags := flag.NewFlagSet("session close", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return fmt.Errorf("usage: whisk session close [-url http://127.0.0.1:8787] <session-id>")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := client.NewHTTP(*baseURL, nil).CloseSession(ctx, protocol.CloseSessionRequest{SessionID: flags.Arg(0)}); err != nil {
		return err
	}
	fmt.Println(flags.Arg(0))
	return nil
}

func runSessionPTY(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: whisk session pty <list|output|tail|kill>")
	}
	switch args[0] {
	case "list":
		return runSessionPTYList(args[1:])
	case "output":
		return runSessionPTYOutput(args[1:])
	case "tail":
		return runSessionPTYTail(args[1:])
	case "kill":
		return runSessionPTYKill(args[1:])
	default:
		return fmt.Errorf("usage: whisk session pty <list|output|tail|kill>")
	}
}

func runSessionPTYList(args []string) error {
	flags := flag.NewFlagSet("session pty list", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() > 1 {
		return fmt.Errorf("usage: whisk session pty list [-url http://127.0.0.1:8787] [session-id]")
	}
	sessionID := ""
	if flags.NArg() == 1 {
		sessionID = flags.Arg(0)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ptys, err := client.NewHTTP(*baseURL, nil).ListPTYs(ctx)
	if err != nil {
		return err
	}
	if sessionID != "" {
		filtered := ptys[:0]
		for _, pty := range ptys {
			if pty.SessionID == sessionID {
				filtered = append(filtered, pty)
			}
		}
		ptys = filtered
	}
	printPTYs(ptys)
	return nil
}

func runSessionPTYOutput(args []string) error {
	flags := flag.NewFlagSet("session pty output", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	fromValue := flags.String("from", "0", "replay from byte offset or end")
	outputJSON := flags.Bool("json", false, "write JSON output")
	plain := flags.Bool("plain", false, "strip terminal control sequences")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return fmt.Errorf("usage: whisk session pty output [-url http://127.0.0.1:8787] [-from offset|end] [-plain] [-json] <pty-id>")
	}
	fromOffset, err := parseOutputOffset(*fromValue)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	snapshot, err := client.NewHTTP(*baseURL, nil).Output(ctx, protocol.OutputRequest{
		PtyID:      flags.Arg(0),
		FromOffset: fromOffset,
	})
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(snapshot)
	}
	return writeOutputSnapshot(snapshot, *plain)
}

func runSessionPTYTail(args []string) error {
	flags := flag.NewFlagSet("session pty tail", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	fromValue := flags.String("from", "end", "tail from byte offset or end")
	pollInterval := flags.Duration("poll", 500*time.Millisecond, "poll interval")
	plain := flags.Bool("plain", false, "strip terminal control sequences")
	once := flags.Bool("once", false, "fetch once and exit")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return fmt.Errorf("usage: whisk session pty tail [-url http://127.0.0.1:8787] [-from offset|end] [-poll 500ms] [-plain] [-once] <pty-id>")
	}
	if *pollInterval <= 0 {
		return fmt.Errorf("poll interval must be positive")
	}
	fromOffset, err := parseOutputOffset(*fromValue)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	daemon := client.NewHTTP(*baseURL, nil)
	offset := fromOffset
	for {
		requestCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		snapshot, err := daemon.Output(requestCtx, protocol.OutputRequest{
			PtyID:      flags.Arg(0),
			FromOffset: offset,
		})
		cancel()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		if err := writeOutputSnapshot(snapshot, *plain); err != nil {
			return err
		}
		if snapshot.Offset > offset {
			offset = snapshot.Offset
		}
		if *once {
			return nil
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(*pollInterval):
		}
	}
}

func parseOutputOffset(value string) (uint64, error) {
	if value == "end" {
		return ^uint64(0), nil
	}
	offset, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid from offset %q", value)
	}
	return offset, nil
}

func writeOutputSnapshot(snapshot protocol.OutputSnapshot, plain bool) error {
	var output []byte
	if snapshot.OutputBase64 != "" {
		decoded, err := base64.StdEncoding.DecodeString(snapshot.OutputBase64)
		if err != nil {
			return err
		}
		output = decoded
	} else {
		output = []byte(snapshot.Output)
	}
	if plain {
		output = stripTerminalControls(output)
	}
	_, err := os.Stdout.Write(output)
	return err
}

func stripTerminalControls(input []byte) []byte {
	out := make([]byte, 0, len(input))
	for i := 0; i < len(input); i++ {
		b := input[i]
		switch b {
		case 0x1b:
			i = skipEscapeSequence(input, i)
		case '\r':
			if i+1 < len(input) && input[i+1] == '\n' {
				out = append(out, '\n')
				i++
			}
		case '\n', '\t':
			out = append(out, b)
		default:
			if b >= 0x20 && b != 0x7f {
				out = append(out, b)
			}
		}
	}
	return out
}

func skipEscapeSequence(input []byte, start int) int {
	if start+1 >= len(input) {
		return start
	}
	switch input[start+1] {
	case '[':
		for i := start + 2; i < len(input); i++ {
			if input[i] >= 0x40 && input[i] <= 0x7e {
				return i
			}
		}
		return len(input) - 1
	case ']':
		for i := start + 2; i < len(input); i++ {
			if input[i] == 0x07 {
				return i
			}
			if input[i] == 0x1b && i+1 < len(input) && input[i+1] == '\\' {
				return i + 1
			}
		}
		return len(input) - 1
	default:
		return start + 1
	}
}

func runSessionPTYKill(args []string) error {
	flags := flag.NewFlagSet("session pty kill", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return fmt.Errorf("usage: whisk session pty kill [-url http://127.0.0.1:8787] <pty-id>")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := client.NewHTTP(*baseURL, nil).KillPTY(ctx, protocol.KillPTYRequest{PTYID: flags.Arg(0)})
	if err != nil {
		return err
	}
	fmt.Printf("%s\t%s\n", result.ID, result.Status)
	return nil
}

func printSessions(sessions []session.Session) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "ID\tPROJECT\tNAME\tROOT\tPANES")
	for _, session := range sessions {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%d\n", session.ID, session.ProjectID, session.Name, session.RootDir, len(session.Panes))
	}
	writer.Flush()
}

func printPTYs(ptys []protocol.PTYInfo) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "ID\tSTATUS\tSESSION\tPANE\tDIR")
	for _, pty := range ptys {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", pty.ID, pty.Status, pty.SessionID, pty.PaneID, pty.WorkingDir)
	}
	writer.Flush()
}
