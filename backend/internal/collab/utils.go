// Package collab - Utility functions for collaboration system
package collab

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strings"
)

// generateID generates a random unique ID
func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// generateInitials generates 2-character initials from a name
func generateInitials(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "??"
	}

	parts := strings.Fields(name)
	if len(parts) == 0 {
		return "??"
	}

	if len(parts) == 1 {
		// Single name - use first two characters
		if len(parts[0]) >= 2 {
			return strings.ToUpper(parts[0][:2])
		}
		return strings.ToUpper(parts[0] + "?")
	}

	// Multiple names - use first letter of first two parts
	return strings.ToUpper(string(parts[0][0]) + string(parts[1][0]))
}

// generateColor generates a consistent color from a user ID
func generateColor(userID string) string {
	// Use hash of ID to generate consistent color
	hash := 0
	for _, c := range userID {
		hash = int(c) + ((hash << 5) - hash)
	}

	// Generate pleasant color from predefined palette
	colors := []string{
		"#667eea", // Blue Purple
		"#764ba2", // Purple
		"#f093fb", // Pink Purple
		"#4facfe", // Sky Blue
		"#00f2fe", // Cyan
		"#43e97b", // Green
		"#38f9d7", // Turquoise
		"#fa709a", // Pink
		"#fee140", // Yellow
		"#ffa647", // Orange
		"#fe8c00", // Dark Orange
		"#f83600", // Red Orange
		"#a8edea", // Light Cyan
		"#fed6e3", // Light Pink
		"#c471f5", // Purple
		"#fa71cd", // Magenta
	}

	idx := abs(hash) % len(colors)
	return colors[idx]
}

// abs returns absolute value of an integer
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// average calculates the average of a float64 slice
func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// percentile calculates the percentile of a float64 slice
func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Sort values
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	// Calculate index
	idx := int(math.Ceil(float64(len(sorted)) * p))
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}

	return sorted[idx]
}

// GenerateSessionURL generates a shareable session URL
func GenerateSessionURL(baseURL string, sessionID string) string {
	if baseURL == "" {
		baseURL = "https://genomevedic.ai"
	}

	return fmt.Sprintf("%s/session/%s", baseURL, sessionID)
}

// ValidatePermission checks if a permission is valid
func ValidatePermission(permission Permission) bool {
	return permission == PermissionOwner ||
		permission == PermissionEditor ||
		permission == PermissionViewer
}

// CanEdit checks if a user can edit based on permission
func CanEdit(permission Permission) bool {
	return permission == PermissionOwner || permission == PermissionEditor
}

// CanPresent checks if a user can control presentation mode
func CanPresent(permission Permission) bool {
	return permission == PermissionOwner
}

// ThrottleCursor determines if a cursor update should be sent based on timing
func ThrottleCursor(lastUpdate int64, nowMs int64) bool {
	const minUpdateIntervalMs = 33 // ~30 Hz
	return (nowMs - lastUpdate) >= minUpdateIntervalMs
}

// ExtractMentions extracts @mentions from markdown content
func ExtractMentions(content string) []string {
	mentions := make([]string, 0)
	words := strings.Fields(content)

	for _, word := range words {
		if strings.HasPrefix(word, "@") {
			username := strings.TrimPrefix(word, "@")
			// Remove trailing punctuation
			username = strings.TrimRight(username, ",.!?;:")
			if username != "" {
				mentions = append(mentions, username)
			}
		}
	}

	return mentions
}

// SanitizeMarkdown performs basic markdown sanitization
func SanitizeMarkdown(content string) string {
	// Remove potentially dangerous HTML
	content = strings.ReplaceAll(content, "<script", "&lt;script")
	content = strings.ReplaceAll(content, "</script>", "&lt;/script&gt;")
	content = strings.ReplaceAll(content, "<iframe", "&lt;iframe")
	content = strings.ReplaceAll(content, "javascript:", "")

	return content
}
