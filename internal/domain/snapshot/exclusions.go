package snapshot

import (
	"errors"
	"slices"
	"strings"
)

const IgnoreFileName = ".stowmarkignore"

var ErrInvalidExcludePattern = errors.New("invalid exclude pattern")

type Exclusions struct {
	patterns        []string
	skipIgnoreFiles bool
}

func NewExclusions(patterns []string, skipIgnoreFiles bool) Exclusions {
	cleaned := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		if pattern = strings.TrimSpace(pattern); pattern != "" && !slices.Contains(cleaned, pattern) {
			cleaned = append(cleaned, pattern)
		}
	}

	return Exclusions{patterns: cleaned, skipIgnoreFiles: skipIgnoreFiles}
}

func ParseExcludePatterns(text string) []string {
	return NewExclusions(strings.Split(text, ","), false).patterns
}

func (e Exclusions) Patterns() []string {
	return slices.Clone(e.patterns)
}

func (e Exclusions) SkipsIgnoreFiles() bool {
	return e.skipIgnoreFiles
}

func (e Exclusions) Empty() bool {
	return len(e.patterns) == 0 && !e.skipIgnoreFiles
}

type SkippedFile struct {
	Path string
	Kind string
}

func SkippedKinds(files []SkippedFile) map[string]int {
	kinds := make(map[string]int)
	for _, file := range files {
		kinds[file.Kind]++
	}

	return kinds
}
