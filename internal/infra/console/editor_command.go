package console

import (
	"fmt"
	"os"
	"os/exec"
)

func editorCommand(filePath string) (*exec.Cmd, error) {
	candidates := []string{
		os.Getenv("VISUAL"),
		os.Getenv("EDITOR"),
		"nvim",
		"vim",
		"nano",
		"vi",
	}

	for _, editor := range candidates {
		if editor == "" {
			continue
		}

		editorPath, err := exec.LookPath(editor)
		if err != nil {
			continue
		}

		return exec.Command(editorPath, filePath), nil
	}

	return nil, fmt.Errorf(
		"no s'ha trobat cap editor; configura $EDITOR o $VISUAL",
	)
}
