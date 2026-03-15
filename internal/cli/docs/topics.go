package docs

import (
	"embed"
	"strings"
)

//go:embed topics/*.md
var topicsFS embed.FS

type topicEntry struct {
	Slug        string
	Description string
	Content     string
}

var topicRegistry []topicEntry

func init() {
	entries, err := topicsFS.ReadDir("topics")
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		slug := strings.TrimSuffix(name, ".md")
		data, err := topicsFS.ReadFile("topics/" + name)
		if err != nil {
			continue
		}
		content := string(data)
		desc := extractDescription(content)
		topicRegistry = append(topicRegistry, topicEntry{
			Slug:        slug,
			Description: desc,
			Content:     content,
		})
	}
}

// extractDescription pulls the first heading from the markdown content.
func extractDescription(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return ""
}

func topicSlugs() []string {
	slugs := make([]string, 0, len(topicRegistry))
	for _, t := range topicRegistry {
		slugs = append(slugs, t.Slug)
	}
	return slugs
}

func findTopic(slug string) (topicEntry, bool) {
	normalized := strings.ToLower(strings.TrimSpace(slug))
	for _, t := range topicRegistry {
		if t.Slug == normalized {
			return t, true
		}
	}
	return topicEntry{}, false
}
