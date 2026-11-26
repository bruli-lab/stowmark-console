package console

import (
	"bytes"
	"fmt"
	"mime"
	"net/http"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

var binaryMediaPrefixes = []string{"image/", "audio/", "video/"}

var binaryMediaTypes = []string{"application/pdf", "application/zip", "application/gzip"}

func isText(content []byte) bool {
	if len(content) == 0 {
		return true
	}

	if bytes.ContainsRune(content, '\x00') {
		return false
	}

	mediaType, _, err := mime.ParseMediaType(http.DetectContentType(content))
	if err == nil && isBinaryMediaType(mediaType) {
		return false
	}

	return validUTF8Prefix(content)
}

func isBinaryMediaType(mediaType string) bool {
	for _, prefix := range binaryMediaPrefixes {
		if strings.HasPrefix(mediaType, prefix) {
			return true
		}
	}

	return slices.Contains(binaryMediaTypes, mediaType)
}

func validUTF8Prefix(content []byte) bool {
	for cut := 0; cut < utf8.UTFMax && cut < len(content); cut++ {
		if utf8.Valid(content[:len(content)-cut]) {
			return true
		}
	}

	return false
}

func truncate(value string, width int) string {
	return ansi.Truncate(value, max(0, width), "…")
}

func truncateLeft(value string, width int) string {
	if ansi.StringWidth(value) <= width {
		return value
	}

	return "…" + ansi.TruncateLeft(value, ansi.StringWidth(value)-width+1, "")
}

func padLeft(value string, width int) string {
	value = truncate(value, width)
	return strings.Repeat(" ", max(0, width-ansi.StringWidth(value))) + value
}

func formatSize(size int64) string {
	const unit = 1024

	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	divisor := int64(unit)
	exponent := 0

	for value := size / unit; value >= unit; value /= unit {
		divisor *= unit
		exponent++
	}

	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(divisor), "KMGTPE"[exponent])
}
