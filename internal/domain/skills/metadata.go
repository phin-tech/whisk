// Package skills holds the daemon-owned skill discovery read-model rules.
package skills

import (
	"strings"
	"unicode"
)

// Metadata is the display metadata parsed from a SKILL.md file.
type Metadata struct {
	Name        string
	Description string
}

type frontmatterValue struct {
	stringValue string
}

// SummarizeMarkdown extracts the skill name and description from SKILL.md
// frontmatter, falling back to the first markdown heading and paragraph.
func SummarizeMarkdown(markdown string) Metadata {
	normalized := strings.TrimPrefix(strings.ReplaceAll(markdown, "\r\n", "\n"), "\ufeff")
	frontmatter, body := splitFrontmatter(normalized)
	values := parseYAMLSubset(frontmatter)

	name := strings.TrimSpace(values["name"].stringValue)
	if name == "" {
		name = firstHeading(body)
	}
	description := strings.TrimSpace(values["description"].stringValue)
	if description == "" {
		description = firstParagraph(body)
	}
	return Metadata{Name: name, Description: description}
}

func splitFrontmatter(markdown string) (string, string) {
	lines := strings.Split(markdown, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", markdown
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return strings.Join(lines[1:i], "\n"), strings.Join(lines[i+1:], "\n")
		}
	}
	return "", markdown
}

func parseYAMLSubset(raw string) map[string]frontmatterValue {
	lines := strings.Split(raw, "\n")
	data := map[string]frontmatterValue{}
	for index := 0; index < len(lines); {
		key, value, ok := parseKeyValueLine(lines[index])
		if !ok {
			index++
			continue
		}

		value = strings.TrimSpace(value)
		switch value {
		case "|", "|-", ">", ">-":
			block := []string{}
			index++
			for index < len(lines) && isIndentedOrBlank(lines[index]) {
				block = append(block, strings.TrimPrefix(lines[index], "  "))
				index++
			}
			data[key] = frontmatterValue{stringValue: collapseWhitespace(strings.Join(block, blockJoiner(value)))}
			continue
		case "":
			foundList := false
			index++
			for index < len(lines) {
				ok := isListItem(lines[index])
				if !ok {
					break
				}
				foundList = true
				index++
			}
			if foundList {
				data[key] = frontmatterValue{}
				continue
			}
			data[key] = frontmatterValue{}
			continue
		default:
			data[key] = frontmatterValue{stringValue: stripQuotePair(value)}
			index++
		}
	}
	return data
}

func parseKeyValueLine(line string) (string, string, bool) {
	colon := strings.IndexByte(line, ':')
	if colon <= 0 {
		return "", "", false
	}
	key := line[:colon]
	for _, r := range key {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-') {
			return "", "", false
		}
	}
	return key, line[colon+1:], true
}

func isListItem(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "-")
}

func isIndentedOrBlank(line string) bool {
	return strings.TrimSpace(line) == "" || strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t")
}

func blockJoiner(style string) string {
	if strings.HasPrefix(style, ">") {
		return " "
	}
	return "\n"
}

func collapseWhitespace(value string) string {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return ""
	}
	return strings.Join(fields, " ")
}

func stripQuotePair(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) < 2 {
		return trimmed
	}
	if (trimmed[0] == '"' && trimmed[len(trimmed)-1] == '"') ||
		(trimmed[0] == '\'' && trimmed[len(trimmed)-1] == '\'') {
		return trimmed[1 : len(trimmed)-1]
	}
	return trimmed
}

func firstHeading(body string) string {
	for _, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if heading, ok := strings.CutPrefix(trimmed, "# "); ok {
			return strings.TrimSpace(heading)
		}
	}
	return ""
}

func firstParagraph(body string) string {
	paragraph := []string{}
	for _, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "```") {
			if len(paragraph) > 0 {
				break
			}
			continue
		}
		paragraph = append(paragraph, trimmed)
		if len(strings.Join(paragraph, " ")) > 240 {
			break
		}
	}
	if len(paragraph) == 0 {
		return ""
	}
	return strings.Join(paragraph, " ")
}
