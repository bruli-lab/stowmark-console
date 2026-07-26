package disk

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/bruli-lab/stowmark-console.git/internal/domain/repository"
)

type FolderRepository struct{}

func (f FolderRepository) IsTextFile(ctx context.Context, filePath string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	file, err := os.Open(filePath)
	if err != nil {
		return false, fmt.Errorf("open file %q: %w", filePath, err)
	}
	defer func() { _ = file.Close() }()

	buffer := make([]byte, 8192)

	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("read file %q: %w", filePath, err)
	}

	if n == 0 {
		return true, nil
	}

	content := buffer[:n]
	contentType := http.DetectContentType(content)

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err == nil {
		if strings.HasPrefix(mediaType, "image/") ||
			strings.HasPrefix(mediaType, "audio/") ||
			strings.HasPrefix(mediaType, "video/") ||
			mediaType == "application/pdf" ||
			mediaType == "application/zip" ||
			mediaType == "application/gzip" {
			return false, nil
		}
	}

	if bytes.ContainsRune(content, '\x00') {
		return false, nil
	}

	return utf8.Valid(content), nil
}

func (f FolderRepository) ReadEntries(ctx context.Context, path string) ([]repository.Entry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed reading repository %q: %w", path, err)
	}

	entries := make([]repository.Entry, len(dirEntries))

	for i, dirEntry := range dirEntries {
		var size int64
		if !dirEntry.IsDir() {
			info, err := dirEntry.Info()
			if err != nil {
				return nil, fmt.Errorf(
					"reading information at %q: %w",
					dirEntry.Name(),
					err,
				)
			}
			size = info.Size()
		}
		entry := repository.NewEntry(dirEntry.Name(), dirEntry.IsDir(), size)
		entries[i] = *entry
	}
	return entries, nil
}

func NewFolderRepository() *FolderRepository {
	return &FolderRepository{}
}
