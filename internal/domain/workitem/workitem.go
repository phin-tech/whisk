package workitem

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	StageKindBacklog  = "backlog"
	StageKindReady    = "ready"
	StageKindActive   = "active"
	StageKindReview   = "review"
	StageKindDone     = "done"
	StageKindArchived = "archived"

	RunStateIdle          = "idle"
	RunStateQueued        = "queued"
	RunStateRunning       = "running"
	RunStateAwaitingInput = "awaiting_input"
	RunStateFailed        = "failed"
	RunStateCompleted     = "completed"
	RunStateCancelled     = "cancelled"

	AttachmentKindFile = "file"
	AttachmentKindURL  = "url"
	AttachmentKindNote = "note"

	AttachmentScopeProject  = "project"
	AttachmentScopeWorktree = "worktree"
	AttachmentScopeExternal = "external"

	HistoryCreated          = "created"
	HistoryStageMoved       = "stage_moved"
	HistoryAttachmentAdded  = "attachment_added"
	HistoryWorktreeBound    = "worktree_bound"
	HistoryDeleted          = "deleted"
	HistoryRunEventAppended = "run_event_appended"

	RunPresetReader   = "reader"
	RunPresetWriter   = "writer"
	RunPresetReviewer = "reviewer"
	RunPresetManager  = "manager"

	PromptTemplatePlan      = "plan"
	PromptTemplateImplement = "implement"
	PromptTemplateReview    = "review"
)

type Project struct {
	ID                 string          `json:"id"`
	Name               string          `json:"name"`
	Slug               string          `json:"slug"`
	RootDir            string          `json:"rootDir"`
	Workflow           ProjectWorkflow `json:"workflow"`
	NextWorkItemNumber int             `json:"nextWorkItemNumber"`
	CreatedAt          time.Time       `json:"createdAt"`
	UpdatedAt          time.Time       `json:"updatedAt"`
}

type WorkflowTemplate struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Source          string           `json:"source"`
	Stages          []WorkflowStage  `json:"stages"`
	TransitionRules []TransitionRule `json:"transitionRules"`
	CreatedAt       time.Time        `json:"createdAt"`
	UpdatedAt       time.Time        `json:"updatedAt"`
}

type ProjectWorkflow struct {
	ID              string           `json:"id"`
	TemplateID      string           `json:"templateId"`
	Name            string           `json:"name"`
	Stages          []WorkflowStage  `json:"stages"`
	TransitionRules []TransitionRule `json:"transitionRules"`
}

type WorkflowStage struct {
	ID                      string `json:"id"`
	Name                    string `json:"name"`
	Kind                    string `json:"kind"`
	DefaultRunPreset        string `json:"defaultRunPreset,omitempty"`
	DefaultPromptTemplateID string `json:"defaultPromptTemplateId,omitempty"`
	ProvisionWorktree       bool   `json:"provisionWorktree,omitempty"`
	WIPLimit                int    `json:"wipLimit,omitempty"`
}

type TransitionRule struct {
	FromStageID           string `json:"fromStageId"`
	ToStageID             string `json:"toStageId"`
	RequiresApproval      bool   `json:"requiresApproval,omitempty"`
	RequiresChecks        bool   `json:"requiresChecks,omitempty"`
	RequiresNoRunningRuns bool   `json:"requiresNoRunningRuns,omitempty"`
}

type WorkItem struct {
	ID           string           `json:"id"`
	ProjectID    string           `json:"projectId"`
	Number       int              `json:"number"`
	Title        string           `json:"title"`
	BodyMarkdown string           `json:"bodyMarkdown"`
	StageID      string           `json:"stageId"`
	RunState     string           `json:"runState"`
	Worktree     *WorktreeBinding `json:"worktree,omitempty"`
	Attachments  []Attachment     `json:"attachments"`
	History      []HistoryEvent   `json:"history"`
	CreatedAt    time.Time        `json:"createdAt"`
	UpdatedAt    time.Time        `json:"updatedAt"`
}

type PromptTemplate struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Source    string    `json:"source"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type WorkItemRun struct {
	ID               string     `json:"id"`
	WorkItemID       string     `json:"workItemId"`
	ProjectID        string     `json:"projectId"`
	Preset           string     `json:"preset"`
	PromptTemplateID string     `json:"promptTemplateId"`
	PromptSnapshot   string     `json:"promptSnapshot"`
	SessionID        string     `json:"sessionId,omitempty"`
	PTYID            string     `json:"ptyId,omitempty"`
	Status           string     `json:"status"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
	CompletedAt      *time.Time `json:"completedAt,omitempty"`
	History          []RunEvent `json:"history"`
}

type RunEvent struct {
	ID      string    `json:"id"`
	Type    string    `json:"type"`
	At      time.Time `json:"at"`
	Actor   string    `json:"actor,omitempty"`
	Message string    `json:"message,omitempty"`
}

type WorktreeBinding struct {
	Branch       string    `json:"branch"`
	Base         string    `json:"base"`
	WorktreePath string    `json:"worktreePath"`
	CreatedAt    time.Time `json:"createdAt"`
}

type Attachment struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Scope     string    `json:"scope"`
	Path      string    `json:"path,omitempty"`
	URL       string    `json:"url,omitempty"`
	Note      string    `json:"note,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type HistoryEvent struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	At           time.Time `json:"at"`
	Actor        string    `json:"actor,omitempty"`
	Message      string    `json:"message,omitempty"`
	StageID      string    `json:"stageId,omitempty"`
	AttachmentID string    `json:"attachmentId,omitempty"`
	Branch       string    `json:"branch,omitempty"`
	WorktreePath string    `json:"worktreePath,omitempty"`
}

type State struct {
	projects        map[string]Project
	templates       map[string]WorkflowTemplate
	promptTemplates map[string]PromptTemplate
	items           map[string]WorkItem
	runs            map[string]WorkItemRun
}

type Snapshot struct {
	Projects        []Project          `json:"projects"`
	Templates       []WorkflowTemplate `json:"workflowTemplates"`
	PromptTemplates []PromptTemplate   `json:"promptTemplates"`
	Items           []WorkItem         `json:"workItems"`
	Runs            []WorkItemRun      `json:"workItemRuns"`
}

type CreateProject struct {
	ID                 string
	WorkflowID         string
	ProjectWorkflowID  string
	Name               string
	Slug               string
	RootDir            string
	NextWorkItemNumber int
	Now                time.Time
}

type CreateWorkItem struct {
	ID           string
	HistoryID    string
	ProjectID    string
	Title        string
	BodyMarkdown string
	StageID      string
	Actor        string
	Now          time.Time
}

type MoveWorkItem struct {
	ID        string
	HistoryID string
	StageID   string
	Actor     string
	Now       time.Time
}

type BindWorktree struct {
	ID           string
	HistoryID    string
	Branch       string
	Base         string
	WorktreePath string
	Actor        string
	Now          time.Time
}

type AddAttachment struct {
	ID         string
	HistoryID  string
	WorkItemID string
	Kind       string
	Scope      string
	Path       string
	URL        string
	Note       string
	Actor      string
	Now        time.Time
}

type DeleteWorkItem struct {
	ID        string
	HistoryID string
	Actor     string
	Now       time.Time
}

type StartRun struct {
	ID               string
	HistoryID        string
	RunHistoryID     string
	WorkItemID       string
	Preset           string
	PromptTemplateID string
	SessionID        string
	PTYID            string
	Actor            string
	Now              time.Time
}

type CancelRun struct {
	ID           string
	RunHistoryID string
	Actor        string
	Now          time.Time
}

type MarkRunRunning struct {
	ID           string
	RunHistoryID string
	SessionID    string
	PTYID        string
	Actor        string
	Now          time.Time
}

type FailRun struct {
	ID           string
	RunHistoryID string
	Actor        string
	Message      string
	Now          time.Time
}

func NewState() *State {
	state := &State{
		projects:        map[string]Project{},
		templates:       map[string]WorkflowTemplate{},
		promptTemplates: map[string]PromptTemplate{},
		items:           map[string]WorkItem{},
		runs:            map[string]WorkItemRun{},
	}
	template := DefaultWorkflowTemplate(time.Time{})
	state.templates[template.ID] = template
	for _, prompt := range DefaultPromptTemplates(time.Time{}) {
		state.promptTemplates[prompt.ID] = prompt
	}
	return state
}

func NewStateFromSnapshot(snapshot Snapshot) (*State, error) {
	state := &State{
		projects:        map[string]Project{},
		templates:       map[string]WorkflowTemplate{},
		promptTemplates: map[string]PromptTemplate{},
		items:           map[string]WorkItem{},
		runs:            map[string]WorkItemRun{},
	}
	if len(snapshot.Templates) == 0 {
		template := DefaultWorkflowTemplate(time.Time{})
		state.templates[template.ID] = template
	}
	if len(snapshot.PromptTemplates) == 0 {
		for _, prompt := range DefaultPromptTemplates(time.Time{}) {
			state.promptTemplates[prompt.ID] = prompt
		}
	}
	for _, template := range snapshot.Templates {
		if err := validateWorkflowTemplate(template); err != nil {
			return nil, err
		}
		if _, exists := state.templates[template.ID]; exists {
			return nil, fmt.Errorf("workflow template %s already exists", template.ID)
		}
		state.templates[template.ID] = cloneTemplate(template)
	}
	for _, prompt := range snapshot.PromptTemplates {
		if err := validatePromptTemplate(prompt); err != nil {
			return nil, err
		}
		if _, exists := state.promptTemplates[prompt.ID]; exists {
			return nil, fmt.Errorf("prompt template %s already exists", prompt.ID)
		}
		state.promptTemplates[prompt.ID] = prompt
	}
	for _, project := range snapshot.Projects {
		if err := validateProject(project); err != nil {
			return nil, err
		}
		if _, exists := state.projects[project.ID]; exists {
			return nil, fmt.Errorf("project %s already exists", project.ID)
		}
		state.projects[project.ID] = cloneProject(project)
	}
	for _, item := range snapshot.Items {
		if err := state.validateWorkItem(item); err != nil {
			return nil, err
		}
		if _, exists := state.items[item.ID]; exists {
			return nil, fmt.Errorf("work item %s already exists", item.ID)
		}
		state.items[item.ID] = cloneWorkItem(item)
	}
	for _, run := range snapshot.Runs {
		if err := state.validateRun(run); err != nil {
			return nil, err
		}
		if _, exists := state.runs[run.ID]; exists {
			return nil, fmt.Errorf("work item run %s already exists", run.ID)
		}
		state.runs[run.ID] = cloneRun(run)
	}
	return state, nil
}

func DefaultWorkflowTemplate(now time.Time) WorkflowTemplate {
	stages := []WorkflowStage{
		{ID: "backlog", Name: "Backlog", Kind: StageKindBacklog, DefaultRunPreset: "reader", DefaultPromptTemplateID: "plan"},
		{ID: "ready", Name: "Ready", Kind: StageKindReady, DefaultRunPreset: "manager", DefaultPromptTemplateID: "plan", ProvisionWorktree: true},
		{ID: "in_progress", Name: "In Progress", Kind: StageKindActive, DefaultRunPreset: "writer", DefaultPromptTemplateID: "implement"},
		{ID: "review", Name: "Review", Kind: StageKindReview, DefaultRunPreset: "reviewer", DefaultPromptTemplateID: "review"},
		{ID: "done", Name: "Done", Kind: StageKindDone},
		{ID: "archived", Name: "Archived", Kind: StageKindArchived},
	}
	return WorkflowTemplate{
		ID:        "default",
		Name:      "Default",
		Source:    "builtin",
		Stages:    stages,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func DefaultPromptTemplates(now time.Time) []PromptTemplate {
	return []PromptTemplate{
		{
			ID:        PromptTemplatePlan,
			Name:      "Plan",
			Source:    "builtin",
			Body:      "Plan the work item.\n\nProject: {{project.name}}\nRoot: {{project.rootDir}}\nWork item: #{{work_item.number}} {{work_item.title}}\n\n{{work_item.bodyMarkdown}}\n\nAttachments:\n{{attachments}}\n",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        PromptTemplateImplement,
			Name:      "Implement",
			Source:    "builtin",
			Body:      "Implement the work item.\n\nProject: {{project.name}}\nRoot: {{project.rootDir}}\nWorktree: {{worktree.path}}\nBranch: {{worktree.branch}}\nWork item: #{{work_item.number}} {{work_item.title}}\n\n{{work_item.bodyMarkdown}}\n\nAttachments:\n{{attachments}}\n",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        PromptTemplateReview,
			Name:      "Review",
			Source:    "builtin",
			Body:      "Review the work item output.\n\nProject: {{project.name}}\nRoot: {{project.rootDir}}\nWorktree: {{worktree.path}}\nBranch: {{worktree.branch}}\nWork item: #{{work_item.number}} {{work_item.title}}\n\n{{work_item.bodyMarkdown}}\n\nAttachments:\n{{attachments}}\n",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

func (s *State) Snapshot() Snapshot {
	return Snapshot{
		Projects:        s.ListProjects(),
		Templates:       s.ListWorkflowTemplates(),
		PromptTemplates: s.ListPromptTemplates(),
		Items:           s.ListWorkItems(""),
		Runs:            s.ListRuns(""),
	}
}

func (s *State) ListProjects() []Project {
	out := make([]Project, 0, len(s.projects))
	for _, project := range s.projects {
		out = append(out, cloneProject(project))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Slug < out[j].Slug
	})
	return out
}

func (s *State) ListWorkflowTemplates() []WorkflowTemplate {
	out := make([]WorkflowTemplate, 0, len(s.templates))
	for _, template := range s.templates {
		out = append(out, cloneTemplate(template))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func (s *State) ListPromptTemplates() []PromptTemplate {
	out := make([]PromptTemplate, 0, len(s.promptTemplates))
	for _, prompt := range s.promptTemplates {
		out = append(out, prompt)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func (s *State) ListWorkItems(projectID string) []WorkItem {
	out := make([]WorkItem, 0, len(s.items))
	for _, item := range s.items {
		if projectID == "" || item.ProjectID == projectID {
			out = append(out, cloneWorkItem(item))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ProjectID != out[j].ProjectID {
			return out[i].ProjectID < out[j].ProjectID
		}
		if out[i].Number != out[j].Number {
			return out[i].Number < out[j].Number
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func (s *State) ListRuns(workItemID string) []WorkItemRun {
	out := make([]WorkItemRun, 0, len(s.runs))
	for _, run := range s.runs {
		if workItemID == "" || run.WorkItemID == workItemID {
			out = append(out, cloneRun(run))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].WorkItemID != out[j].WorkItemID {
			return out[i].WorkItemID < out[j].WorkItemID
		}
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func (s *State) GetProject(id string) (Project, bool) {
	project, ok := s.projects[id]
	return cloneProject(project), ok
}

func (s *State) GetWorkItem(id string) (WorkItem, bool) {
	item, ok := s.items[id]
	return cloneWorkItem(item), ok
}

func (s *State) CreateProject(req CreateProject) (Project, error) {
	if req.ID == "" {
		return Project{}, fmt.Errorf("project id required")
	}
	if _, exists := s.projects[req.ID]; exists {
		return Project{}, fmt.Errorf("project %s already exists", req.ID)
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return Project{}, fmt.Errorf("project name required")
	}
	rootDir, err := cleanAbsolutePath(req.RootDir, "root dir")
	if err != nil {
		return Project{}, err
	}
	slug := cleanSlug(req.Slug)
	if slug == "" {
		slug = cleanSlug(name)
	}
	if slug == "" {
		return Project{}, fmt.Errorf("project slug required")
	}
	workflowID := req.WorkflowID
	if workflowID == "" {
		workflowID = "default"
	}
	template, ok := s.templates[workflowID]
	if !ok {
		return Project{}, fmt.Errorf("workflow template %s not found", workflowID)
	}
	projectWorkflowID := req.ProjectWorkflowID
	if projectWorkflowID == "" {
		projectWorkflowID = req.ID + "-workflow"
	}
	nextNumber := req.NextWorkItemNumber
	if nextNumber <= 0 {
		nextNumber = 1
	}
	project := Project{
		ID:      req.ID,
		Name:    name,
		Slug:    slug,
		RootDir: rootDir,
		Workflow: ProjectWorkflow{
			ID:              projectWorkflowID,
			TemplateID:      template.ID,
			Name:            template.Name,
			Stages:          cloneStages(template.Stages),
			TransitionRules: cloneTransitionRules(template.TransitionRules),
		},
		NextWorkItemNumber: nextNumber,
		CreatedAt:          req.Now,
		UpdatedAt:          req.Now,
	}
	if err := validateProject(project); err != nil {
		return Project{}, err
	}
	s.projects[project.ID] = project
	return cloneProject(project), nil
}

func (s *State) CreateWorkItem(req CreateWorkItem) (WorkItem, error) {
	if req.ID == "" {
		return WorkItem{}, fmt.Errorf("work item id required")
	}
	if _, exists := s.items[req.ID]; exists {
		return WorkItem{}, fmt.Errorf("work item %s already exists", req.ID)
	}
	project, ok := s.projects[req.ProjectID]
	if !ok {
		return WorkItem{}, fmt.Errorf("project %s not found", req.ProjectID)
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return WorkItem{}, fmt.Errorf("work item title required")
	}
	stageID := req.StageID
	if stageID == "" {
		stageID = project.Workflow.Stages[0].ID
	}
	if !project.hasStage(stageID) {
		return WorkItem{}, fmt.Errorf("stage %s not found", stageID)
	}
	stage, _ := project.stage(stageID)
	if stage.ProvisionWorktree {
		return WorkItem{}, fmt.Errorf("stage %s requires worktree", stageID)
	}
	number := project.NextWorkItemNumber
	project.NextWorkItemNumber++
	project.UpdatedAt = req.Now
	item := WorkItem{
		ID:           req.ID,
		ProjectID:    project.ID,
		Number:       number,
		Title:        title,
		BodyMarkdown: req.BodyMarkdown,
		StageID:      stageID,
		RunState:     RunStateIdle,
		CreatedAt:    req.Now,
		UpdatedAt:    req.Now,
		History: []HistoryEvent{{
			ID:      req.HistoryID,
			Type:    HistoryCreated,
			At:      req.Now,
			Actor:   req.Actor,
			Message: "created work item",
			StageID: stageID,
		}},
	}
	if item.History[0].ID == "" {
		return WorkItem{}, fmt.Errorf("history id required")
	}
	s.projects[project.ID] = project
	s.items[item.ID] = item
	return cloneWorkItem(item), nil
}

func (s *State) MoveWorkItem(req MoveWorkItem) (WorkItem, error) {
	item, ok := s.items[req.ID]
	if !ok {
		return WorkItem{}, fmt.Errorf("work item %s not found", req.ID)
	}
	project := s.projects[item.ProjectID]
	stage, ok := project.stage(req.StageID)
	if !ok {
		return WorkItem{}, fmt.Errorf("stage %s not found", req.StageID)
	}
	if stage.ProvisionWorktree && item.Worktree == nil {
		return WorkItem{}, fmt.Errorf("stage %s requires worktree", req.StageID)
	}
	item.StageID = req.StageID
	item.UpdatedAt = req.Now
	if err := appendHistory(&item, HistoryEvent{
		ID:      req.HistoryID,
		Type:    HistoryStageMoved,
		At:      req.Now,
		Actor:   req.Actor,
		Message: "moved work item",
		StageID: req.StageID,
	}); err != nil {
		return WorkItem{}, err
	}
	s.items[item.ID] = item
	return cloneWorkItem(item), nil
}

func (s *State) BindWorktree(req BindWorktree) (WorkItem, error) {
	item, ok := s.items[req.ID]
	if !ok {
		return WorkItem{}, fmt.Errorf("work item %s not found", req.ID)
	}
	if item.Worktree != nil {
		return WorkItem{}, fmt.Errorf("work item %s already has worktree", req.ID)
	}
	branch := strings.TrimSpace(req.Branch)
	if branch == "" {
		return WorkItem{}, fmt.Errorf("branch required")
	}
	worktreePath, err := cleanAbsolutePath(req.WorktreePath, "worktree path")
	if err != nil {
		return WorkItem{}, err
	}
	item.Worktree = &WorktreeBinding{
		Branch:       branch,
		Base:         strings.TrimSpace(req.Base),
		WorktreePath: worktreePath,
		CreatedAt:    req.Now,
	}
	item.UpdatedAt = req.Now
	if err := appendHistory(&item, HistoryEvent{
		ID:           req.HistoryID,
		Type:         HistoryWorktreeBound,
		At:           req.Now,
		Actor:        req.Actor,
		Message:      "bound worktree",
		Branch:       branch,
		WorktreePath: worktreePath,
	}); err != nil {
		return WorkItem{}, err
	}
	s.items[item.ID] = item
	return cloneWorkItem(item), nil
}

func (s *State) AddAttachment(req AddAttachment) (WorkItem, error) {
	item, ok := s.items[req.WorkItemID]
	if !ok {
		return WorkItem{}, fmt.Errorf("work item %s not found", req.WorkItemID)
	}
	attachment, err := validateAttachment(Attachment{
		ID:        req.ID,
		Kind:      req.Kind,
		Scope:     req.Scope,
		Path:      req.Path,
		URL:       req.URL,
		Note:      req.Note,
		CreatedAt: req.Now,
	})
	if err != nil {
		return WorkItem{}, err
	}
	item.Attachments = append(item.Attachments, attachment)
	item.UpdatedAt = req.Now
	if err := appendHistory(&item, HistoryEvent{
		ID:           req.HistoryID,
		Type:         HistoryAttachmentAdded,
		At:           req.Now,
		Actor:        req.Actor,
		Message:      "added attachment",
		AttachmentID: attachment.ID,
	}); err != nil {
		return WorkItem{}, err
	}
	s.items[item.ID] = item
	return cloneWorkItem(item), nil
}

func (s *State) DeleteWorkItem(req DeleteWorkItem) (WorkItem, error) {
	item, ok := s.items[req.ID]
	if !ok {
		return WorkItem{}, fmt.Errorf("work item %s not found", req.ID)
	}
	if err := appendHistory(&item, HistoryEvent{
		ID:      req.HistoryID,
		Type:    HistoryDeleted,
		At:      req.Now,
		Actor:   req.Actor,
		Message: "deleted work item",
	}); err != nil {
		return WorkItem{}, err
	}
	delete(s.items, req.ID)
	return cloneWorkItem(item), nil
}

func (s *State) StartRun(req StartRun) (WorkItemRun, error) {
	if req.ID == "" {
		return WorkItemRun{}, fmt.Errorf("run id required")
	}
	if _, exists := s.runs[req.ID]; exists {
		return WorkItemRun{}, fmt.Errorf("work item run %s already exists", req.ID)
	}
	item, ok := s.items[req.WorkItemID]
	if !ok {
		return WorkItemRun{}, fmt.Errorf("work item %s not found", req.WorkItemID)
	}
	project := s.projects[item.ProjectID]
	preset := strings.TrimSpace(req.Preset)
	if preset == "" {
		preset = defaultPresetForStage(project, item.StageID)
	}
	if !validPreset(preset) {
		return WorkItemRun{}, fmt.Errorf("unsupported run preset %s", preset)
	}
	templateID := strings.TrimSpace(req.PromptTemplateID)
	if templateID == "" {
		templateID = defaultPromptForStage(project, item.StageID)
	}
	template, ok := s.promptTemplates[templateID]
	if !ok {
		return WorkItemRun{}, fmt.Errorf("prompt template %s not found", templateID)
	}
	prompt := renderPrompt(template.Body, project, item)
	run := WorkItemRun{
		ID:               req.ID,
		WorkItemID:       item.ID,
		ProjectID:        item.ProjectID,
		Preset:           preset,
		PromptTemplateID: template.ID,
		PromptSnapshot:   prompt,
		SessionID:        strings.TrimSpace(req.SessionID),
		PTYID:            strings.TrimSpace(req.PTYID),
		Status:           RunStateQueued,
		CreatedAt:        req.Now,
		UpdatedAt:        req.Now,
		History: []RunEvent{{
			ID:      req.RunHistoryID,
			Type:    RunStateQueued,
			At:      req.Now,
			Actor:   req.Actor,
			Message: "started work item run",
		}},
	}
	if run.History[0].ID == "" {
		return WorkItemRun{}, fmt.Errorf("run history id required")
	}
	if err := appendHistory(&item, HistoryEvent{
		ID:      req.HistoryID,
		Type:    HistoryRunEventAppended,
		At:      req.Now,
		Actor:   req.Actor,
		Message: "started work item run",
	}); err != nil {
		return WorkItemRun{}, err
	}
	item.RunState = RunStateQueued
	item.UpdatedAt = req.Now
	s.items[item.ID] = item
	s.runs[run.ID] = run
	return cloneRun(run), nil
}

func (s *State) CancelRun(req CancelRun) (WorkItemRun, error) {
	run, ok := s.runs[req.ID]
	if !ok {
		return WorkItemRun{}, fmt.Errorf("work item run %s not found", req.ID)
	}
	if run.Status == RunStateCompleted || run.Status == RunStateFailed || run.Status == RunStateCancelled {
		return WorkItemRun{}, fmt.Errorf("work item run %s is already terminal", req.ID)
	}
	run.Status = RunStateCancelled
	run.UpdatedAt = req.Now
	run.CompletedAt = &req.Now
	if err := appendRunEvent(&run, RunEvent{
		ID:      req.RunHistoryID,
		Type:    RunStateCancelled,
		At:      req.Now,
		Actor:   req.Actor,
		Message: "cancelled work item run",
	}); err != nil {
		return WorkItemRun{}, err
	}
	item := s.items[run.WorkItemID]
	item.RunState = RunStateCancelled
	item.UpdatedAt = req.Now
	s.items[item.ID] = item
	s.runs[run.ID] = run
	return cloneRun(run), nil
}

func (s *State) MarkRunRunning(req MarkRunRunning) (WorkItemRun, error) {
	run, ok := s.runs[req.ID]
	if !ok {
		return WorkItemRun{}, fmt.Errorf("work item run %s not found", req.ID)
	}
	if run.Status != RunStateQueued {
		return WorkItemRun{}, fmt.Errorf("work item run %s is %s, not queued", req.ID, run.Status)
	}
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return WorkItemRun{}, fmt.Errorf("session id required")
	}
	ptyID := strings.TrimSpace(req.PTYID)
	if ptyID == "" {
		return WorkItemRun{}, fmt.Errorf("pty id required")
	}
	run.SessionID = sessionID
	run.PTYID = ptyID
	run.Status = RunStateRunning
	run.UpdatedAt = req.Now
	if err := appendRunEvent(&run, RunEvent{
		ID:      req.RunHistoryID,
		Type:    RunStateRunning,
		At:      req.Now,
		Actor:   req.Actor,
		Message: "launched work item run",
	}); err != nil {
		return WorkItemRun{}, err
	}
	item := s.items[run.WorkItemID]
	item.RunState = RunStateRunning
	item.UpdatedAt = req.Now
	s.items[item.ID] = item
	s.runs[run.ID] = run
	return cloneRun(run), nil
}

func (s *State) FailRun(req FailRun) (WorkItemRun, error) {
	run, ok := s.runs[req.ID]
	if !ok {
		return WorkItemRun{}, fmt.Errorf("work item run %s not found", req.ID)
	}
	if run.Status == RunStateCompleted || run.Status == RunStateFailed || run.Status == RunStateCancelled {
		return WorkItemRun{}, fmt.Errorf("work item run %s is already terminal", req.ID)
	}
	message := strings.TrimSpace(req.Message)
	if message == "" {
		message = "work item run failed"
	}
	run.Status = RunStateFailed
	run.UpdatedAt = req.Now
	run.CompletedAt = &req.Now
	if err := appendRunEvent(&run, RunEvent{
		ID:      req.RunHistoryID,
		Type:    RunStateFailed,
		At:      req.Now,
		Actor:   req.Actor,
		Message: message,
	}); err != nil {
		return WorkItemRun{}, err
	}
	item := s.items[run.WorkItemID]
	item.RunState = RunStateFailed
	item.UpdatedAt = req.Now
	s.items[item.ID] = item
	s.runs[run.ID] = run
	return cloneRun(run), nil
}

func (p Project) hasStage(stageID string) bool {
	_, ok := p.stage(stageID)
	return ok
}

func defaultPresetForStage(project Project, stageID string) string {
	if stage, ok := project.stage(stageID); ok && stage.DefaultRunPreset != "" {
		return stage.DefaultRunPreset
	}
	return RunPresetReader
}

func defaultPromptForStage(project Project, stageID string) string {
	if stage, ok := project.stage(stageID); ok && stage.DefaultPromptTemplateID != "" {
		return stage.DefaultPromptTemplateID
	}
	return PromptTemplatePlan
}

func (p Project) stage(stageID string) (WorkflowStage, bool) {
	for _, stage := range p.Workflow.Stages {
		if stage.ID == stageID {
			return stage, true
		}
	}
	return WorkflowStage{}, false
}

func (s *State) validateWorkItem(item WorkItem) error {
	if item.ID == "" {
		return fmt.Errorf("work item id required")
	}
	project, ok := s.projects[item.ProjectID]
	if !ok {
		return fmt.Errorf("project %s not found", item.ProjectID)
	}
	if item.Number <= 0 {
		return fmt.Errorf("work item number must be positive")
	}
	if strings.TrimSpace(item.Title) == "" {
		return fmt.Errorf("work item title required")
	}
	if !project.hasStage(item.StageID) {
		return fmt.Errorf("stage %s not found", item.StageID)
	}
	if item.RunState == "" {
		return fmt.Errorf("work item run state required")
	}
	for _, attachment := range item.Attachments {
		if _, err := validateAttachment(attachment); err != nil {
			return err
		}
	}
	return nil
}

func validateWorkflowTemplate(template WorkflowTemplate) error {
	if template.ID == "" {
		return fmt.Errorf("workflow template id required")
	}
	if strings.TrimSpace(template.Name) == "" {
		return fmt.Errorf("workflow template name required")
	}
	if len(template.Stages) == 0 {
		return fmt.Errorf("workflow template stages required")
	}
	seen := map[string]struct{}{}
	for _, stage := range template.Stages {
		if stage.ID == "" {
			return fmt.Errorf("workflow stage id required")
		}
		if strings.TrimSpace(stage.Name) == "" {
			return fmt.Errorf("workflow stage name required")
		}
		if stage.Kind == "" {
			return fmt.Errorf("workflow stage kind required")
		}
		if _, exists := seen[stage.ID]; exists {
			return fmt.Errorf("workflow stage %s already exists", stage.ID)
		}
		seen[stage.ID] = struct{}{}
	}
	return nil
}

func validatePromptTemplate(template PromptTemplate) error {
	if template.ID == "" {
		return fmt.Errorf("prompt template id required")
	}
	if strings.TrimSpace(template.Name) == "" {
		return fmt.Errorf("prompt template name required")
	}
	if strings.TrimSpace(template.Body) == "" {
		return fmt.Errorf("prompt template body required")
	}
	return nil
}

func validateProject(project Project) error {
	if project.ID == "" {
		return fmt.Errorf("project id required")
	}
	if strings.TrimSpace(project.Name) == "" {
		return fmt.Errorf("project name required")
	}
	if cleanSlug(project.Slug) == "" {
		return fmt.Errorf("project slug required")
	}
	if _, err := cleanAbsolutePath(project.RootDir, "root dir"); err != nil {
		return err
	}
	if project.NextWorkItemNumber <= 0 {
		return fmt.Errorf("next work item number must be positive")
	}
	return validateWorkflowTemplate(WorkflowTemplate{
		ID:     project.Workflow.ID,
		Name:   project.Workflow.Name,
		Stages: project.Workflow.Stages,
	})
}

func validateAttachment(attachment Attachment) (Attachment, error) {
	if attachment.ID == "" {
		return Attachment{}, fmt.Errorf("attachment id required")
	}
	if attachment.Kind == "" {
		return Attachment{}, fmt.Errorf("attachment kind required")
	}
	if attachment.Scope == "" {
		attachment.Scope = AttachmentScopeProject
	}
	switch attachment.Kind {
	case AttachmentKindFile:
		if attachment.Path == "" {
			return Attachment{}, fmt.Errorf("attachment path required")
		}
		if filepath.IsAbs(attachment.Path) && attachment.Scope != AttachmentScopeExternal {
			return Attachment{}, fmt.Errorf("absolute attachment path requires external scope")
		}
		attachment.Path = filepath.Clean(attachment.Path)
	case AttachmentKindURL:
		if strings.TrimSpace(attachment.URL) == "" {
			return Attachment{}, fmt.Errorf("attachment url required")
		}
	case AttachmentKindNote:
		if strings.TrimSpace(attachment.Note) == "" {
			return Attachment{}, fmt.Errorf("attachment note required")
		}
	default:
		return Attachment{}, fmt.Errorf("unsupported attachment kind %s", attachment.Kind)
	}
	return attachment, nil
}

func appendHistory(item *WorkItem, event HistoryEvent) error {
	if event.ID == "" {
		return fmt.Errorf("history id required")
	}
	if event.Type == "" {
		return fmt.Errorf("history event type required")
	}
	item.History = append(item.History, event)
	return nil
}

func appendRunEvent(run *WorkItemRun, event RunEvent) error {
	if event.ID == "" {
		return fmt.Errorf("run history id required")
	}
	if event.Type == "" {
		return fmt.Errorf("run event type required")
	}
	run.History = append(run.History, event)
	return nil
}

func (s *State) validateRun(run WorkItemRun) error {
	if run.ID == "" {
		return fmt.Errorf("work item run id required")
	}
	item, ok := s.items[run.WorkItemID]
	if !ok {
		return fmt.Errorf("work item %s not found", run.WorkItemID)
	}
	if run.ProjectID != item.ProjectID {
		return fmt.Errorf("work item run project mismatch")
	}
	if !validPreset(run.Preset) {
		return fmt.Errorf("unsupported run preset %s", run.Preset)
	}
	if _, ok := s.promptTemplates[run.PromptTemplateID]; !ok {
		return fmt.Errorf("prompt template %s not found", run.PromptTemplateID)
	}
	if strings.TrimSpace(run.PromptSnapshot) == "" {
		return fmt.Errorf("prompt snapshot required")
	}
	if run.Status == "" {
		return fmt.Errorf("run status required")
	}
	return nil
}

func validPreset(preset string) bool {
	switch preset {
	case RunPresetReader, RunPresetWriter, RunPresetReviewer, RunPresetManager:
		return true
	default:
		return false
	}
}

func renderPrompt(template string, project Project, item WorkItem) string {
	replacements := map[string]string{
		"{{project.name}}":           project.Name,
		"{{project.rootDir}}":        project.RootDir,
		"{{work_item.id}}":           item.ID,
		"{{work_item.number}}":       fmt.Sprintf("%d", item.Number),
		"{{work_item.title}}":        item.Title,
		"{{work_item.bodyMarkdown}}": item.BodyMarkdown,
		"{{work_item.stageId}}":      item.StageID,
		"{{attachments}}":            renderAttachments(item.Attachments),
		"{{worktree.branch}}":        "",
		"{{worktree.path}}":          "",
	}
	if item.Worktree != nil {
		replacements["{{worktree.branch}}"] = item.Worktree.Branch
		replacements["{{worktree.path}}"] = item.Worktree.WorktreePath
	}
	out := template
	for old, replacement := range replacements {
		out = strings.ReplaceAll(out, old, replacement)
	}
	return strings.TrimSpace(out)
}

func renderAttachments(attachments []Attachment) string {
	if len(attachments) == 0 {
		return "- none"
	}
	lines := make([]string, 0, len(attachments))
	for _, attachment := range attachments {
		value := attachment.Path
		if value == "" {
			value = attachment.URL
		}
		if value == "" {
			value = attachment.Note
		}
		lines = append(lines, "- "+value)
	}
	return strings.Join(lines, "\n")
}

func cleanAbsolutePath(path string, label string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("%s required", label)
	}
	cleaned := filepath.Clean(path)
	if !filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("%s must be absolute", label)
	}
	return cleaned, nil
}

func cleanSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || r == ' ' || r == '.':
			if !lastDash && builder.Len() > 0 {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(builder.String(), "-")
}

func cloneProject(project Project) Project {
	project.Workflow.Stages = cloneStages(project.Workflow.Stages)
	project.Workflow.TransitionRules = cloneTransitionRules(project.Workflow.TransitionRules)
	return project
}

func cloneTemplate(template WorkflowTemplate) WorkflowTemplate {
	template.Stages = cloneStages(template.Stages)
	template.TransitionRules = cloneTransitionRules(template.TransitionRules)
	return template
}

func cloneWorkItem(item WorkItem) WorkItem {
	if item.Worktree != nil {
		worktree := *item.Worktree
		item.Worktree = &worktree
	}
	item.Attachments = append([]Attachment(nil), item.Attachments...)
	item.History = append([]HistoryEvent(nil), item.History...)
	return item
}

func cloneRun(run WorkItemRun) WorkItemRun {
	if run.CompletedAt != nil {
		completedAt := *run.CompletedAt
		run.CompletedAt = &completedAt
	}
	run.History = append([]RunEvent(nil), run.History...)
	return run
}

func cloneStages(stages []WorkflowStage) []WorkflowStage {
	return append([]WorkflowStage(nil), stages...)
}

func cloneTransitionRules(rules []TransitionRule) []TransitionRule {
	return append([]TransitionRule(nil), rules...)
}
