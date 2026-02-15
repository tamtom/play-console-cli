package notify

import (
	"fmt"
	"strings"
	"time"
)

// PayloadFormat represents a webhook payload format.
type PayloadFormat string

const (
	FormatSlack   PayloadFormat = "slack"
	FormatDiscord PayloadFormat = "discord"
	FormatGeneric PayloadFormat = "generic"
)

// ParseFormat validates and returns a PayloadFormat.
func ParseFormat(s string) (PayloadFormat, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "slack", "":
		return FormatSlack, nil
	case "discord":
		return FormatDiscord, nil
	case "generic":
		return FormatGeneric, nil
	default:
		return "", fmt.Errorf("unsupported format: %q (supported: slack, discord, generic)", s)
	}
}

// SlackPayload is the JSON structure for Slack webhooks.
type SlackPayload struct {
	Text   string       `json:"text"`
	Blocks []SlackBlock `json:"blocks,omitempty"`
}

// SlackBlock represents a Slack block element.
type SlackBlock struct {
	Type string         `json:"type"`
	Text *SlackTextObj  `json:"text,omitempty"`
}

// SlackTextObj is a Slack text object used inside blocks.
type SlackTextObj struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// DiscordPayload is the JSON structure for Discord webhooks.
type DiscordPayload struct {
	Content string `json:"content"`
}

// GenericPayload is the JSON structure for generic HTTP webhooks.
type GenericPayload struct {
	EventType string `json:"event_type,omitempty"`
	Package   string `json:"package,omitempty"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// BuildPayload constructs the appropriate payload for the given format.
func BuildPayload(format PayloadFormat, message, eventType, packageName string) interface{} {
	switch format {
	case FormatDiscord:
		return buildDiscordPayload(message, eventType, packageName)
	case FormatGeneric:
		return buildGenericPayload(message, eventType, packageName)
	default:
		return buildSlackPayload(message, eventType, packageName)
	}
}

func buildSlackPayload(message, eventType, packageName string) SlackPayload {
	body := formatMessageBody(message, eventType, packageName)
	return SlackPayload{
		Text: message,
		Blocks: []SlackBlock{
			{
				Type: "section",
				Text: &SlackTextObj{
					Type: "mrkdwn",
					Text: body,
				},
			},
		},
	}
}

func buildDiscordPayload(message, eventType, packageName string) DiscordPayload {
	body := formatMessageBody(message, eventType, packageName)
	return DiscordPayload{
		Content: body,
	}
}

func buildGenericPayload(message, eventType, packageName string) GenericPayload {
	return GenericPayload{
		EventType: eventType,
		Package:   packageName,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// formatMessageBody builds a human-readable message string with optional metadata.
func formatMessageBody(message, eventType, packageName string) string {
	var parts []string

	if strings.TrimSpace(eventType) != "" {
		parts = append(parts, fmt.Sprintf("[%s]", eventType))
	}
	if strings.TrimSpace(packageName) != "" {
		parts = append(parts, fmt.Sprintf("`%s`", packageName))
	}

	if len(parts) > 0 {
		return fmt.Sprintf("%s %s", strings.Join(parts, " "), message)
	}
	return message
}
