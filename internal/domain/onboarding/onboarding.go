package onboarding

const (
	StatusCurrent     = "current"
	StatusMissing     = "missing"
	StatusOutdated    = "outdated"
	StatusModified    = "modified"
	StatusUntrusted   = "untrusted"
	StatusUnavailable = "unavailable"

	KindDaemon = "daemon"
	KindHook   = "hook"
	KindSkill  = "skill"
	KindPlugin = "plugin"
)

type Item struct {
	ID                string `json:"id"`
	Kind              string `json:"kind"`
	Label             string `json:"label"`
	Description       string `json:"description,omitempty"`
	Target            string `json:"target"`
	Status            string `json:"status"`
	SelectedByDefault bool   `json:"selectedByDefault"`
	Version           string `json:"version,omitempty"`
	InstalledVersion  string `json:"installedVersion,omitempty"`
	LatestVersion     string `json:"latestVersion,omitempty"`
	Hash              string `json:"hash,omitempty"`
	InstalledHash     string `json:"installedHash,omitempty"`
	Path              string `json:"path,omitempty"`
	Detail            string `json:"detail,omitempty"`
}

type Record struct {
	Status  string `json:"status,omitempty"`
	Version string `json:"version,omitempty"`
	Hash    string `json:"hash,omitempty"`
}

type State struct {
	Applied          map[string]Record `json:"applied,omitempty"`
	Skipped          map[string]bool   `json:"skipped,omitempty"`
	DaemonAPIVersion int               `json:"daemonApiVersion,omitempty"`
	DaemonGitSHA     string            `json:"daemonGitSha,omitempty"`
}

func SelectDefaults(items []Item) []Item {
	out := append([]Item(nil), items...)
	for i := range out {
		out[i].SelectedByDefault = defaultSelected(out[i])
	}
	return out
}

func ShouldShow(items []Item, state State) bool {
	if len(state.Applied) == 0 && len(state.Skipped) == 0 {
		for _, item := range items {
			if actionable(item) {
				return true
			}
		}
		return false
	}
	for _, item := range items {
		if state.Skipped[item.ID] || !actionable(item) {
			continue
		}
		if record, ok := state.Applied[item.ID]; ok && changed(item, record) {
			return true
		}
	}
	return false
}

func NextState(current State, items []Item, selected map[string]bool, apiVersion int, gitSHA string) State {
	next := State{
		Applied:          copyRecords(current.Applied),
		Skipped:          copyBools(current.Skipped),
		DaemonAPIVersion: apiVersion,
		DaemonGitSHA:     gitSHA,
	}
	if next.Applied == nil {
		next.Applied = map[string]Record{}
	}
	if next.Skipped == nil {
		next.Skipped = map[string]bool{}
	}
	for _, item := range items {
		if selected[item.ID] {
			delete(next.Skipped, item.ID)
			next.Applied[item.ID] = Record{Status: item.Status, Version: item.LatestVersion, Hash: item.Hash}
			continue
		}
		if _, known := next.Applied[item.ID]; !known {
			next.Skipped[item.ID] = true
		}
	}
	return next
}

func defaultSelected(item Item) bool {
	if item.Kind == KindPlugin || item.Status == StatusModified || item.Status == StatusUnavailable || item.Status == StatusCurrent {
		return false
	}
	return item.Status == StatusMissing || item.Status == StatusOutdated || item.Status == StatusUntrusted
}

func actionable(item Item) bool {
	return item.Status == StatusMissing ||
		item.Status == StatusOutdated ||
		item.Status == StatusModified ||
		item.Status == StatusUntrusted
}

func changed(item Item, record Record) bool {
	if record.Status == "" {
		return true
	}
	return record.Status != item.Status ||
		(record.Version != "" && record.Version != item.LatestVersion) ||
		(record.Hash != "" && record.Hash != item.Hash)
}

func copyRecords(in map[string]Record) map[string]Record {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]Record, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func copyBools(in map[string]bool) map[string]bool {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]bool, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
