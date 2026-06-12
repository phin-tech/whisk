package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/phin-tech/whisk/internal/client"
	"github.com/phin-tech/whisk/internal/protocol"
)

func runWorkflow(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: whisk workflow <start-planning|submit-plan|approve-plan|start-execution|complete-execution|feedback>")
	}
	switch args[0] {
	case "start-planning":
		return runWorkflowStartPlanning(args[1:])
	case "submit-plan":
		return runWorkflowSubmitPlan(args[1:])
	case "approve-plan":
		return runWorkflowApprovePlan(args[1:])
	case "start-execution":
		return runWorkflowStartExecution(args[1:])
	case "complete-execution":
		return runWorkflowCompleteExecution(args[1:])
	case "feedback":
		return runWorkflowFeedback(args[1:])
	default:
		return fmt.Errorf("usage: whisk workflow <start-planning|submit-plan|approve-plan|start-execution|complete-execution|feedback>")
	}
}

func runWorkflowStartPlanning(args []string) error {
	flags := workflowFlagSet("workflow start-planning")
	baseURL, outputJSON, workItemID, actor := workflowCommonFlags(flags)
	launch := flags.Bool("launch", false, "launch agent PTY")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *workItemID == "" {
		return fmt.Errorf("usage: whisk workflow start-planning [-work-item id] [-actor actor] [-launch] [-json] [-url url]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	run, err := client.NewHTTP(*baseURL, nil).StartPlanning(ctx, protocol.StartPlanningRequest{
		WorkItemID: *workItemID,
		Launch:     *launch,
		Actor:      *actor,
	})
	if err != nil {
		return err
	}
	return printMaybeJSON(run, *outputJSON)
}

func runWorkflowSubmitPlan(args []string) error {
	flags := workflowFlagSet("workflow submit-plan")
	baseURL, outputJSON, workItemID, actor := workflowCommonFlags(flags)
	runID := flags.String("run", envOrDefault("WHISK_RUN_ID", ""), "run id")
	title := flags.String("title", "Plan", "plan title")
	body := flags.String("body", "", "plan body")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *workItemID == "" || *body == "" {
		return fmt.Errorf("usage: whisk workflow submit-plan -body text [-work-item id] [-run id] [-title title] [-actor actor] [-json] [-url url]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	artifact, err := client.NewHTTP(*baseURL, nil).SubmitDraftPlan(ctx, protocol.SubmitDraftPlanRequest{
		WorkItemID: *workItemID,
		RunID:      *runID,
		Title:      *title,
		Body:       *body,
		Actor:      *actor,
	})
	if err != nil {
		return err
	}
	return printMaybeJSON(artifact, *outputJSON)
}

func runWorkflowApprovePlan(args []string) error {
	flags := workflowFlagSet("workflow approve-plan")
	baseURL, outputJSON, workItemID, actor := workflowCommonFlags(flags)
	artifactID := flags.String("artifact", "", "plan artifact id")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *workItemID == "" || *artifactID == "" {
		return fmt.Errorf("usage: whisk workflow approve-plan -artifact id [-work-item id] [-actor actor] [-json] [-url url]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	item, err := client.NewHTTP(*baseURL, nil).ApprovePlan(ctx, protocol.ApprovePlanRequest{
		WorkItemID: *workItemID,
		ArtifactID: *artifactID,
		Actor:      *actor,
	})
	if err != nil {
		return err
	}
	return printMaybeJSON(item, *outputJSON)
}

func runWorkflowStartExecution(args []string) error {
	flags := workflowFlagSet("workflow start-execution")
	baseURL, outputJSON, workItemID, actor := workflowCommonFlags(flags)
	launch := flags.Bool("launch", false, "launch agent PTY")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *workItemID == "" {
		return fmt.Errorf("usage: whisk workflow start-execution [-work-item id] [-actor actor] [-launch] [-json] [-url url]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	run, err := client.NewHTTP(*baseURL, nil).StartExecution(ctx, protocol.StartExecutionRequest{
		WorkItemID: *workItemID,
		Launch:     *launch,
		Actor:      *actor,
	})
	if err != nil {
		return err
	}
	return printMaybeJSON(run, *outputJSON)
}

func runWorkflowCompleteExecution(args []string) error {
	flags := workflowFlagSet("workflow complete-execution")
	baseURL, outputJSON, _, actor := workflowCommonFlags(flags)
	runID := flags.String("run", envOrDefault("WHISK_RUN_ID", ""), "run id")
	message := flags.String("message", "", "completion message")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *runID == "" {
		return fmt.Errorf("usage: whisk workflow complete-execution [-run id] [-message text] [-actor actor] [-json] [-url url]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	item, err := client.NewHTTP(*baseURL, nil).CompleteExecution(ctx, protocol.CompleteExecutionRequest{
		RunID:   *runID,
		Message: *message,
		Actor:   *actor,
	})
	if err != nil {
		return err
	}
	return printMaybeJSON(item, *outputJSON)
}

func runWorkflowFeedback(args []string) error {
	flags := workflowFlagSet("workflow feedback")
	baseURL, outputJSON, workItemID, actor := workflowCommonFlags(flags)
	runID := flags.String("run", envOrDefault("WHISK_RUN_ID", ""), "run id")
	body := flags.String("body", "", "feedback body")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *workItemID == "" || *body == "" {
		return fmt.Errorf("usage: whisk workflow feedback -body text [-work-item id] [-run id] [-actor actor] [-json] [-url url]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	artifact, err := client.NewHTTP(*baseURL, nil).SubmitReviewFeedback(ctx, protocol.SubmitReviewFeedbackRequest{
		WorkItemID: *workItemID,
		RunID:      *runID,
		Body:       *body,
		Actor:      *actor,
	})
	if err != nil {
		return err
	}
	return printMaybeJSON(artifact, *outputJSON)
}

func runQuestion(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: whisk question <ask|answer>")
	}
	switch args[0] {
	case "ask":
		return runQuestionAsk(args[1:])
	case "answer":
		return runQuestionAnswer(args[1:])
	default:
		return fmt.Errorf("usage: whisk question <ask|answer>")
	}
}

func runQuestionAsk(args []string) error {
	flags := workflowFlagSet("question ask")
	baseURL, outputJSON, workItemID, actor := workflowCommonFlags(flags)
	runID := flags.String("run", envOrDefault("WHISK_RUN_ID", ""), "run id")
	prompt := flags.String("prompt", "", "question prompt")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *workItemID == "" || *prompt == "" {
		return fmt.Errorf("usage: whisk question ask -prompt text [-work-item id] [-run id] [-actor actor] [-json] [-url url]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	question, err := client.NewHTTP(*baseURL, nil).AskQuestion(ctx, protocol.AskQuestionRequest{
		WorkItemID: *workItemID,
		RunID:      *runID,
		Prompt:     *prompt,
		Actor:      *actor,
	})
	if err != nil {
		return err
	}
	return printMaybeJSON(question, *outputJSON)
}

func runQuestionAnswer(args []string) error {
	flags := workflowFlagSet("question answer")
	baseURL, outputJSON, _, actor := workflowCommonFlags(flags)
	answer := flags.String("answer", "", "question answer")
	questionID := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		questionID = args[0]
		args = args[1:]
	}
	if err := flags.Parse(args); err != nil {
		return err
	}
	if questionID == "" && flags.NArg() == 1 {
		questionID = flags.Arg(0)
	}
	if flags.NArg() > 1 || questionID == "" || *answer == "" {
		return fmt.Errorf("usage: whisk question answer <question-id> -answer text [-actor actor] [-json] [-url url]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	question, err := client.NewHTTP(*baseURL, nil).AnswerQuestion(ctx, protocol.AnswerQuestionRequest{
		ID:     questionID,
		Answer: *answer,
		Actor:  *actor,
	})
	if err != nil {
		return err
	}
	return printMaybeJSON(question, *outputJSON)
}

func workflowFlagSet(name string) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	return flags
}

func workflowCommonFlags(flags *flag.FlagSet) (*string, *bool, *string, *string) {
	baseURL := flags.String("url", envOrDefault("WHISKD_URL", "http://127.0.0.1:8787"), "daemon URL")
	outputJSON := flags.Bool("json", false, "write JSON output")
	workItemID := flags.String("work-item", envOrDefault("WHISK_WORK_ITEM_ID", ""), "work item id")
	actor := flags.String("actor", envOrDefault("WHISK_ACTOR", ""), "actor")
	return baseURL, outputJSON, workItemID, actor
}

func printMaybeJSON(value any, outputJSON bool) error {
	if outputJSON {
		return printJSON(value)
	}
	fmt.Printf("%v\n", value)
	return nil
}
