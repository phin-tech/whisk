package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/domain/workitem"
	"github.com/phin-tech/whisk/internal/protocol"
)

func runProject(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: whisk project <list|create|show|update>")
	}
	switch args[0] {
	case "list":
		return runProjectList(args[1:])
	case "create":
		return runProjectCreate(args[1:])
	case "show":
		return runProjectShow(args[1:])
	case "update":
		return runProjectUpdate(args[1:])
	default:
		return fmt.Errorf("usage: whisk project <list|create|show|update>")
	}
}

func runProjectList(args []string) error {
	flags := flag.NewFlagSet("project list", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: whisk project list [-json] [-url http://127.0.0.1:8787]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	projects, err := client.NewHTTP(*baseURL, nil).ListProjects(ctx)
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(projects)
	}
	printProjects(projects)
	return nil
}

func runProjectCreate(args []string) error {
	flags := flag.NewFlagSet("project create", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	name := flags.String("name", "", "project name")
	description := flags.String("description", "", "project description")
	slug := flags.String("slug", "", "project slug")
	rootDir := flags.String("root", "", "project root directory")
	workflowID := flags.String("workflow", "", "workflow template id")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *name == "" || *rootDir == "" {
		return fmt.Errorf("usage: whisk project create -name <name> -root <path> [-description text] [-slug slug] [-workflow id] [-json] [-url http://127.0.0.1:8787]")
	}
	resolvedRoot, err := filepath.Abs(*rootDir)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	project, err := client.NewHTTP(*baseURL, nil).CreateProject(ctx, protocol.CreateProjectRequest{
		Name:        *name,
		Description: *description,
		Slug:        *slug,
		RootDir:     resolvedRoot,
		WorkflowID:  *workflowID,
	})
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(project)
	}
	fmt.Println(project.ID)
	return nil
}

func runProjectShow(args []string) error {
	flags := flag.NewFlagSet("project show", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return fmt.Errorf("usage: whisk project show <project-id> [-json] [-url http://127.0.0.1:8787]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	detail, err := client.NewHTTP(*baseURL, nil).GetProjectDetail(ctx, flags.Arg(0))
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(detail)
	}
	fmt.Println(detail.Project.ID)
	return nil
}

func runProjectUpdate(args []string) error {
	flags := flag.NewFlagSet("project update", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	var req protocol.UpdateProjectRequest
	flags.Func("name", "project name", func(value string) error {
		req.Name = &value
		return nil
	})
	flags.Func("description", "project description", func(value string) error {
		req.Description = &value
		return nil
	})
	flags.Func("slug", "project slug", func(value string) error {
		req.Slug = &value
		return nil
	})
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 || (req.Name == nil && req.Description == nil && req.Slug == nil) {
		return fmt.Errorf("usage: whisk project update <project-id> [-name name] [-description text] [-slug slug] [-json] [-url http://127.0.0.1:8787]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	project, err := client.NewHTTP(*baseURL, nil).UpdateProject(ctx, flags.Arg(0), req)
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(project)
	}
	fmt.Println(project.ID)
	return nil
}

func runWorkItem(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: whisk work-item <list|create|move|bind-worktree|attach-file|delete>")
	}
	switch args[0] {
	case "list":
		return runWorkItemList(args[1:])
	case "create":
		return runWorkItemCreate(args[1:])
	case "move":
		return runWorkItemMove(args[1:])
	case "bind-worktree":
		return runWorkItemBindWorktree(args[1:])
	case "attach-file":
		return runWorkItemAttachFile(args[1:])
	case "delete":
		return runWorkItemDelete(args[1:])
	default:
		return fmt.Errorf("usage: whisk work-item <list|create|move|bind-worktree|attach-file|delete>")
	}
}

func runWorkItemList(args []string) error {
	flags := flag.NewFlagSet("work-item list", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	projectID := flags.String("project", envOrDefault("WHISK_PROJECT", ""), "project id")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: whisk work-item list [-project project-id] [-json] [-url http://127.0.0.1:8787]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	items, err := client.NewHTTP(*baseURL, nil).ListWorkItems(ctx, *projectID)
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(items)
	}
	printWorkItems(items)
	return nil
}

func runWorkItemCreate(args []string) error {
	flags := flag.NewFlagSet("work-item create", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	projectID := flags.String("project", envOrDefault("WHISK_PROJECT", ""), "project id")
	workflowID := flags.String("workflow", "", "workflow id")
	title := flags.String("title", "", "work item title")
	body := flags.String("body", "", "markdown body")
	stageID := flags.String("stage", "", "initial stage id")
	actor := flags.String("actor", envOrDefault("WHISK_ACTOR", ""), "actor")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *projectID == "" || *title == "" {
		return fmt.Errorf("usage: whisk work-item create -project <id> -title <title> [-body markdown] [-stage stage] [-actor actor] [-json] [-url http://127.0.0.1:8787]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	item, err := client.NewHTTP(*baseURL, nil).CreateWorkItem(ctx, protocol.CreateWorkItemRequest{
		ProjectID:    *projectID,
		WorkflowID:   *workflowID,
		Title:        *title,
		BodyMarkdown: *body,
		StageID:      *stageID,
		Actor:        *actor,
	})
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(item)
	}
	fmt.Printf("%s\t#%d\n", item.ID, item.Number)
	return nil
}

func runWorkItemMove(args []string) error {
	flags := flag.NewFlagSet("work-item move", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	stageID := flags.String("stage", "", "target stage id")
	actor := flags.String("actor", envOrDefault("WHISK_ACTOR", ""), "actor")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 || *stageID == "" {
		return fmt.Errorf("usage: whisk work-item move -stage <stage> <work-item-id> [-actor actor] [-json] [-url http://127.0.0.1:8787]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	item, err := client.NewHTTP(*baseURL, nil).MoveWorkItem(ctx, protocol.MoveWorkItemRequest{
		ID:      flags.Arg(0),
		StageID: *stageID,
		Actor:   *actor,
	})
	if err != nil {
		return err
	}
	return printWorkItemResult(item, *outputJSON)
}

func runWorkItemBindWorktree(args []string) error {
	flags := flag.NewFlagSet("work-item bind-worktree", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	branch := flags.String("branch", "", "branch name")
	base := flags.String("base", "", "base branch")
	path := flags.String("path", "", "worktree path")
	actor := flags.String("actor", envOrDefault("WHISK_ACTOR", ""), "actor")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 || *branch == "" || *path == "" {
		return fmt.Errorf("usage: whisk work-item bind-worktree -branch <branch> -path <path> <work-item-id> [-base base] [-actor actor] [-json] [-url http://127.0.0.1:8787]")
	}
	resolvedPath, err := filepath.Abs(*path)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	item, err := client.NewHTTP(*baseURL, nil).BindWorkItemWorktree(ctx, protocol.BindWorkItemWorktreeRequest{
		ID:           flags.Arg(0),
		Branch:       *branch,
		Base:         *base,
		WorktreePath: resolvedPath,
		Actor:        *actor,
	})
	if err != nil {
		return err
	}
	return printWorkItemResult(item, *outputJSON)
}

func runWorkItemAttachFile(args []string) error {
	flags := flag.NewFlagSet("work-item attach-file", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	scope := flags.String("scope", workitem.AttachmentScopeProject, "attachment scope")
	actor := flags.String("actor", envOrDefault("WHISK_ACTOR", ""), "actor")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 2 {
		return fmt.Errorf("usage: whisk work-item attach-file <work-item-id> <path> [-scope project|worktree|external] [-actor actor] [-json] [-url http://127.0.0.1:8787]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	item, err := client.NewHTTP(*baseURL, nil).AddWorkItemAttachment(ctx, protocol.AddWorkItemAttachmentRequest{
		WorkItemID: flags.Arg(0),
		Kind:       workitem.AttachmentKindFile,
		Scope:      *scope,
		Path:       flags.Arg(1),
		Actor:      *actor,
	})
	if err != nil {
		return err
	}
	return printWorkItemResult(item, *outputJSON)
}

func runWorkItemDelete(args []string) error {
	flags := flag.NewFlagSet("work-item delete", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	actor := flags.String("actor", envOrDefault("WHISK_ACTOR", ""), "actor")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return fmt.Errorf("usage: whisk work-item delete <work-item-id> [-actor actor] [-json] [-url http://127.0.0.1:8787]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	item, err := client.NewHTTP(*baseURL, nil).DeleteWorkItem(ctx, protocol.DeleteWorkItemRequest{
		ID:    flags.Arg(0),
		Actor: *actor,
	})
	if err != nil {
		return err
	}
	return printWorkItemResult(item, *outputJSON)
}

func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func printProjects(projects []protocol.Project) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "ID\tNAME\tSLUG\tROOT")
	for _, project := range projects {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", project.ID, project.Name, project.Slug, project.RootDir)
	}
	writer.Flush()
}

func printWorkItems(items []protocol.WorkItem) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "ID\tNO\tTITLE\tSTAGE\tRUN\tWORKTREE")
	for _, item := range items {
		worktreePath := ""
		if item.Worktree != nil {
			worktreePath = item.Worktree.WorktreePath
		}
		fmt.Fprintf(writer, "%s\t#%d\t%s\t%s\t%s\t%s\n", item.ID, item.Number, item.Title, item.StageID, item.RunState, worktreePath)
	}
	writer.Flush()
}

func printWorkItemResult(item protocol.WorkItem, outputJSON bool) error {
	if outputJSON {
		return printJSON(item)
	}
	fmt.Printf("%s\t#%d\t%s\n", item.ID, item.Number, item.StageID)
	return nil
}
