package agents

import "os/exec"

type ExecutableLookup func(command string) (string, error)

type DetectedProfile struct {
	ProfileID     string
	Provider      Provider
	Label         string
	DetectCommand string
	Path          string
}

func DetectProfiles(profiles []ProfileInfo, lookup ExecutableLookup) []DetectedProfile {
	if lookup == nil {
		lookup = exec.LookPath
	}
	var detected []DetectedProfile
	for _, profile := range profiles {
		for _, command := range detectCommands(profile) {
			path, err := lookup(command)
			if err != nil || path == "" {
				continue
			}
			detected = append(detected, DetectedProfile{
				ProfileID:     profile.ID,
				Provider:      profile.Provider,
				Label:         profile.Label,
				DetectCommand: command,
				Path:          path,
			})
			break
		}
	}
	return detected
}

func detectCommands(profile ProfileInfo) []string {
	if profile.DetectCmd == "" && len(profile.DetectAliases) == 0 {
		return nil
	}
	commands := make([]string, 0, 1+len(profile.DetectAliases))
	if profile.DetectCmd != "" {
		commands = append(commands, profile.DetectCmd)
	}
	commands = append(commands, profile.DetectAliases...)
	return commands
}
