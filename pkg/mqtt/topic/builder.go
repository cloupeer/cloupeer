package topic

import (
	"strings"
)

// Builder provides generic capabilities for constructing MQTT topics.
// It handles path joining and wildcard appending without specific business logic.
type Builder struct {
	// root is the common prefix for all topics (e.g., "iov/v1").
	root string
}

// NewBuilder creates a new generic Topic Builder.
func NewBuilder(root string) *Builder {
	// Trim trailing slash to prevent double slashes.
	return &Builder{
		root: strings.TrimSuffix(root, "/"),
	}
}

// Build constructs a topic path by joining the root and provided segments.
// Usage:
//
//	b.Build("command", "vh001") -> "root/command/vh001"
//	b.Build("region", topic.Wildcard, "status") -> "root/region/+/status"
func (b *Builder) Build(segments ...string) string {
	// Pre-allocate slice capacity: root + segments.
	parts := make([]string, 0, 1+len(segments))
	parts = append(parts, b.root)
	parts = append(parts, segments...)
	return strings.Join(parts, "/")
}

// BuildWildcard appends segments and a single-level wildcard "+" at the end.
// Usage: b.BuildWildcard("command", "ack") -> "root/command/ack/+"
func (b *Builder) BuildWildcard(segments ...string) string {
	return b.Build(append(segments, Wildcard)...)
}

// BuildMultiWildcard appends segments and a multi-level wildcard "#" at the end.
// Usage: b.BuildMultiWildcard("sys") -> "root/sys/#"
func (b *Builder) BuildMultiWildcard(segments ...string) string {
	return b.Build(append(segments, MultiWildcard)...)
}
