package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/domain/session"
	"github.com/phin-tech/whisk/internal/protocol"
)

func runSession(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: whisk session <list|create|set-root|close|pty>")
	}
	switch args[0] {
	case "list":
		return runSessionList(args[1:])
	case "create":
		return runSessionCreate(args[1:])
	case "set-root":
		return runSessionSetRoot(args[1:])
	case "close":
		return runSessionClose(args[1:])
	case "pty":
		return runSessionPTY(args[1:])
	default:
		return fmt.Errorf("usage: whisk session <list|create|set-root|close|pty>")
	}
}

func runSessionList(args []string) error {
	flags := flag.NewFlagSet("session list", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: whisk session list [-url http://127.0.0.1:8787]")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	sessions, err := client.NewHTTP(*baseURL, nil).ListSessions(ctx)
	if err != nil {
		return err
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
	command := flags.String("command", "", "initial command to run in the PTY shell")
	startPTY := flags.Bool("pty", true, "start an initial PTY")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *rootDir == "" {
		return fmt.Errorf("usage: whisk session create -root <path> [-name name] [-command command] [-pty=false] [-url http://127.0.0.1:8787]")
	}
	resolvedRoot, err := filepath.Abs(*rootDir)
	if err != nil {
		return err
	}

	req := protocol.CreateSessionRequest{
		Name:    *name,
		RootDir: resolvedRoot,
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
		return fmt.Errorf("usage: whisk session pty <list|kill>")
	}
	switch args[0] {
	case "list":
		return runSessionPTYList(args[1:])
	case "kill":
		return runSessionPTYKill(args[1:])
	default:
		return fmt.Errorf("usage: whisk session pty <list|kill>")
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
	fmt.Fprintln(writer, "ID\tNAME\tROOT\tPANES")
	for _, session := range sessions {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%d\n", session.ID, session.Name, session.RootDir, len(session.Panes))
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
