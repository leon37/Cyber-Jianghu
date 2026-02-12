package adapters

import (
	"Cyber-Jianghu/server/internal/interfaces"
	"regexp"
	"strings"
)

// CommandType represents the type of command parsed from danmaku
type CommandType string

const (
	CommandAction CommandType = "action"
	CommandVote  CommandType = "vote"
	CommandNone  CommandType = "none"
)

// ParsedCommand represents a parsed command from danmaku
type ParsedCommand struct {
	Type    CommandType
	Action  string
	Params  map[string]string
	VoteID  string
	RawText string
}

// DanmakuParser parses danmaku content for commands and votes
type DanmakuParser struct {
	actionPattern *regexp.Regexp
	votePattern  *regexp.Regexp
}

// NewDanmakuParser creates a new danmaku parser
func NewDanmakuParser() *DanmakuParser {
	return &DanmakuParser{
		actionPattern: regexp.MustCompile(`^/(\w+)(?:\s+(.+))?$`),
		votePattern:  regexp.MustCompile(`^/vote\s+(\d+)$`),
	}
}

// Parse parses a danmaku message and extracts commands
func (p *DanmakuParser) Parse(danmaku interfaces.Danmaku) *ParsedCommand {
	trimmed := strings.TrimSpace(danmaku.Content)
	result := &ParsedCommand{
		RawText: trimmed,
	}

	// Check for vote command
	if match := p.votePattern.FindStringSubmatch(trimmed); match != nil {
		result.Type = CommandVote
		result.VoteID = match[1]
		return result
	}

	// Check for action command
	if match := p.actionPattern.FindStringSubmatch(trimmed); match != nil {
		result.Type = CommandAction
		result.Action = match[1]

		// Parse parameters
		if len(match) > 2 && match[2] != "" {
			result.Params = parseParams(match[2])
		}
		return result
	}

	result.Type = CommandNone
	return result
}

// parseParams parses command parameters from string
func parseParams(params string) map[string]string {
	result := make(map[string]string)

	// Parse key=value format
	parts := strings.Fields(params)
	for _, part := range parts {
		if idx := strings.Index(part, "="); idx > 0 {
			key := part[:idx]
			value := part[idx+1:]
			result[key] = value
		} else {
			// Positional parameter
			result[strconv.Itoa(len(result))] = part
		}
	}

	return result
}

// IsActionCommand checks if text is an action command
func (p *DanmakuParser) IsActionCommand(text string) bool {
	return p.actionPattern.MatchString(strings.TrimSpace(text))
}

// IsVoteCommand checks if text is a vote command
func (p *DanmakuParser) IsVoteCommand(text string) bool {
	return p.votePattern.MatchString(strings.TrimSpace(text))
}
