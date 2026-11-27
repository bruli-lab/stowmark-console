package console

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	keyExtension     = ".pem"
	privateKeySuffix = "-private.pem"
	publicKeySuffix  = "-public.pem"
	pemHeader        = "-----BEGIN "
)

var (
	errFileNotFound = errors.New("file not found")
	errNotAFile     = errors.New("not a file")
	errNotAPEMKey   = errors.New("not a PEM key file")
)

func defaultKeysFolder() string {
	base, err := os.UserConfigDir()
	if err != nil {
		base = os.TempDir()
	}

	return filepath.Join(base, "stowmark-console", "keys")
}

func describeKeysFolder(value string) (string, error) {
	path, err := resolveLocalPath(value)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "The directory will be created", nil
	case err != nil:
		return "", err
	case !info.IsDir():
		return "", errNotADirectory
	}

	return describeDirectory(value)
}

func describeKeyFile(value string) (string, error) {
	path, err := resolveLocalPath(value)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "", errFileNotFound
	case err != nil:
		return "", err
	case info.IsDir():
		return "", errNotAFile
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	if !bytes.HasPrefix(bytes.TrimSpace(content), []byte(pemHeader)) {
		return "", errNotAPEMKey
	}

	return fmt.Sprintf("PEM key file (%s)", formatSize(info.Size())), nil
}

func pairedKey(path, suffix, pairSuffix string) string {
	if !strings.HasSuffix(path, suffix) {
		return ""
	}

	paired := strings.TrimSuffix(path, suffix) + pairSuffix
	if _, err := os.Stat(paired); err != nil {
		return ""
	}

	return paired
}
