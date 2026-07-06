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
		return fmt.Errorf("usage: whisk workflow <actions|run-action|start-planning|submit-plan|approve-plan|start-execution|complete-execution|feedback|approve-done|artifacts|events>")
	}
	switch args[0] {
	case "actions":
		return runWorkflowActions(args[1:])
	case "run-action":
		return runWorkflowRunAction(args[1:])
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
	case "approve-done":
		return runWorkflowApproveDone(args[1:])
	case "artifacts":
		return runWorkflowArtifacts(args[1:])
	case "events":
		return runWorkflowEvents(args[1:])
	default:
		return fmt.Errorf("usage: whisk workflow <actions|run-action|start-planning|submit-plan|approve-plan|start-execution|complete-execution|feedback|approve-done|artifacts|events>")
	}
}

func runWorkflowActions(args []string) error {
	flags := workflowFlagSet("workflow actions")
	baseURL, outputJSON, workItemID, _ := workflowCommonFlags(flags)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *workItemID == "" {
		return fmt.Errorf("usage: whisk workflow actions [-work-item id] [-json] [-url url]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	actions, err := client.NewHTTP(*baseURL, nil).ListWorkItemWorkflowActions(ctx, *workItemID)
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(actions)
	}
	for _, action := range actions {
		fmt.Printf("%s\t%t\t%s\t%s\n", action.Action.ID, action.Enabled, action.InputKind, action.Reason)
	}
	return nil
}

func runWorkflowRunAction(args []string) error {
	flags := workflowFlagSet("workflow run-action")
	baseURL, outputJSON, workItemID, actor := workflowCommonFlags(flags)
	actionID := flags.String("action", "", "workflow action id")
	runID := flags.String("run", envOrDefault("WHISK_RUN_ID", ""), "run id")
	reason := flags.String("reason", "", "action reason")
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		*actionID = args[0]
		args = args[1:]
	}
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *actionID == "" && flags.NArg() == 1 {
		*actionID = flags.Arg(0)
	}
	if flags.NArg() > 1 || *workItemID == "" || *actionID == "" {
		return fmt.Errorf("usage: whisk workflow run-action <action-id> [-work-item id] [-run id] [-reason text] [-actor actor] [-json] [-url url]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	item, err := client.NewHTTP(*baseURL, nil).RunWorkItemWorkflowAction(ctx, protocol.RunWorkItemWorkflowActionRequest{
		WorkItemID: *workItemID,
		ActionID:   *actionID,
		RunID:      *runID,
		Reason:     *reason,
		Actor:      *actor,
	})
	if err != nil {
		return err
	}
	return printMaybeJSON(item, *outputJSON)
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

func runWorkflowApproveDone(args []string) error {
	flags := workflowFlagSet("workflow approve-done")
	baseURL, outputJSON, workItemID, actor := workflowCommonFlags(flags)
	reason := flags.String("reason", "", "approval reason")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *workItemID == "" {
		return fmt.Errorf("usage: whisk workflow approve-done [-work-item id] [-reason text] [-actor actor] [-json] [-url url]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	item, err := client.NewHTTP(*baseURL, nil).ApproveDone(ctx, protocol.ApproveDoneRequest{
		WorkItemID: *workItemID,
		Reason:     *reason,
		Actor:      *actor,
	})
	if err != nil {
		return err
	}
	return printMaybeJSON(item, *outputJSON)
}

func runWorkflowArtifacts(args []string) error {
	flags := workflowFlagSet("workflow artifacts")
	baseURL, outputJSON, workItemID, _ := workflowCommonFlags(flags)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: whisk workflow artifacts [-work-item id] [-json] [-url url]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	artifacts, err := client.NewHTTP(*baseURL, nil).ListArtifacts(ctx, *workItemID)
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(artifacts)
	}
	for _, artifact := range artifacts {
		fmt.Printf("%s\t%s\t%s\t%s\n", artifact.ID, artifact.WorkItemID, artifact.Kind, artifact.Status)
	}
	return nil
}

func runWorkflowEvents(args []string) error {
	flags := workflowFlagSet("workflow events")
	baseURL, outputJSON, workItemID, _ := workflowCommonFlags(flags)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: whisk workflow events [-work-item id] [-json] [-url url]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	events, err := client.NewHTTP(*baseURL, nil).ListWorkflowEvents(ctx, *workItemID)
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(events)
	}
	for _, event := range events {
		fmt.Printf("%s\t%s\t%s\t%s\n", event.ID, event.WorkItemID, event.Type, event.Actor)
	}
	return nil
}

func runGate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: whisk gate <list|complete>")
	}
	switch args[0] {
	case "list":
		return runGateList(args[1:])
	case "complete":
		return runGateComplete(args[1:])
	default:
		return fmt.Errorf("usage: whisk gate <list|complete>")
	}
}

func runGateList(args []string) error {
	flags := workflowFlagSet("gate list")
	baseURL, outputJSON, workItemID, _ := workflowCommonFlags(flags)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: whisk gate list [-work-item id] [-json] [-url url]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	gates, err := client.NewHTTP(*baseURL, nil).ListGateReports(ctx, *workItemID)
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(gates)
	}
	for _, gate := range gates {
		fmt.Printf("%s\t%s\t%s\t%t\n", gate.ID, gate.WorkItemID, gate.Status, gate.Blocking)
	}
	return nil
}

func runGateComplete(args []string) error {
	flags := workflowFlagSet("gate complete")
	baseURL, outputJSON, _, actor := workflowCommonFlags(flags)
	status := flags.String("status", "", "gate status: passed, failed, or overridden")
	overrideReason := flags.String("override-reason", "", "reason required when status is overridden")
	gateID := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		gateID = args[0]
		args = args[1:]
	}
	if err := flags.Parse(args); err != nil {
		return err
	}
	if gateID == "" && flags.NArg() == 1 {
		gateID = flags.Arg(0)
	}
	if flags.NArg() > 1 || gateID == "" || *status == "" {
		return fmt.Errorf("usage: whisk gate complete <gate-report-id> -status <passed|failed|overridden> [-override-reason text] [-actor actor] [-json] [-url url]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	gate, err := client.NewHTTP(*baseURL, nil).CompleteGate(ctx, protocol.CompleteGateRequest{
		ID:             gateID,
		Status:         *status,
		OverrideReason: *overrideReason,
		Actor:          *actor,
	})
	if err != nil {
		return err
	}
	return printMaybeJSON(gate, *outputJSON)
}

func runQuestion(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: whisk question <list|ask|answer>")
	}
	switch args[0] {
	case "list":
		return runQuestionList(args[1:])
	case "ask":
		return runQuestionAsk(args[1:])
	case "answer":
		return runQuestionAnswer(args[1:])
	default:
		return fmt.Errorf("usage: whisk question <list|ask|answer>")
	}
}

func runQuestionList(args []string) error {
	flags := workflowFlagSet("question list")
	baseURL, outputJSON, workItemID, _ := workflowCommonFlags(flags)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: whisk question list [-work-item id] [-json] [-url url]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	questions, err := client.NewHTTP(*baseURL, nil).ListQuestions(ctx, *workItemID)
	if err != nil {
		return err
	}
	if *outputJSON {
		return printJSON(questions)
	}
	for _, question := range questions {
		fmt.Printf("%s\t%s\t%s\n", question.ID, question.WorkItemID, question.Status)
	}
	return nil
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
