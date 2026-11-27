package console

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
)

const (
	initLabelWidth       = 13
	initLevelWidth       = 4
	initFolderWidth      = 56
	encryptionNone       = "none"
	encryptionNewKeys    = "new key pair"
	encryptionConfigured = "configured public key"
)

type initField int

const (
	initCompression initField = iota
	initLevel
	initFormat
	initEncryption
	initKeysFolder
)

type initForm struct {
	title       string
	location    string
	warning     string
	declineHint string
	publicKey   *string
	compression int
	level       textinput.Model
	format      int
	encryption  int
	encryptions []string
	keysFolder  textinput.Model
	focus       initField
	err         error
}

func newInitForm(profile *repository.Profile, warning, declineHint string) *initForm {
	options := profile.Init()

	form := &initForm{
		title:       "Initialize repository " + profile.Name(),
		location:    profile.Location(),
		warning:     warning,
		declineHint: declineHint,
		publicKey:   options.PublicKey(),
		compression: max(0, slices.Index(repository.Compressions(), options.Compression())),
		level:       newInput("", levelText(options.CompressionLevel()), initLevelWidth),
		format:      options.FormatVersion(),
		encryptions: []string{encryptionNone, encryptionNewKeys},
		keysFolder:  newInput("", defaultKeysFolder(), initFolderWidth),
	}

	if options.Encrypted() {
		form.encryptions = []string{encryptionConfigured}
	}

	form.focusField(initCompression)

	return form
}

func levelText(level *int) string {
	if level == nil {
		return ""
	}

	return strconv.Itoa(*level)
}

func (f *initForm) compressionName() string {
	return repository.Compressions()[f.compression]
}

func (f *initForm) encryptionName() string {
	return f.encryptions[f.encryption]
}

func (f *initForm) visible() []initField {
	fields := []initField{initCompression}
	if _, ok := repository.LevelsOf(f.compressionName()); ok {
		fields = append(fields, initLevel)
	}
	fields = append(fields, initFormat, initEncryption)
	if f.encryptionName() == encryptionNewKeys {
		fields = append(fields, initKeysFolder)
	}

	return fields
}

func (f *initForm) focusField(field initField) {
	f.focus = field
	f.level.Blur()
	f.keysFolder.Blur()

	switch field {
	case initLevel:
		f.level.Focus()
	case initKeysFolder:
		f.keysFolder.Focus()
	case initCompression, initFormat, initEncryption:
	}
}

func (f *initForm) move(step int) {
	fields := f.visible()
	index := max(0, slices.Index(fields, f.focus))
	f.focusField(fields[(index+step+len(fields))%len(fields)])
}

func (f *initForm) cycle(step int) {
	switch f.focus {
	case initCompression:
		count := len(repository.Compressions())
		f.compression = (f.compression + step + count) % count
		f.level.SetValue("")
	case initFormat:
		span := repository.LatestFormatVersion - repository.MinimumFormatVersion + 1
		f.format = (f.format-repository.MinimumFormatVersion+step+span)%span + repository.MinimumFormatVersion
	case initEncryption:
		f.encryption = (f.encryption + step + len(f.encryptions)) % len(f.encryptions)
		if f.encryptionName() == encryptionNewKeys {
			f.format = repository.LatestFormatVersion
		}
	case initLevel, initKeysFolder:
	}
}

func (f *initForm) update(msg tea.Msg) (bool, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "esc", "ctrl+c":
			return true, emit(repositoryInitDeclined{})
		case "tab", "down":
			f.move(1)
			return false, nil
		case "shift+tab", "up":
			f.move(-1)
			return false, nil
		case "enter":
			return f.submit()
		case "left", "right":
			if f.focus != initLevel && f.focus != initKeysFolder {
				f.cycle(map[string]int{"left": -1, "right": 1}[key.String()])
				f.err = nil
				return false, nil
			}
		}
	}

	var cmd tea.Cmd
	switch f.focus {
	case initLevel:
		f.level, cmd = f.level.Update(msg)
	case initKeysFolder:
		f.keysFolder, cmd = f.keysFolder.Update(msg)
	case initCompression, initFormat, initEncryption:
	}
	f.err = nil

	return false, cmd
}

func (f *initForm) options() (repository.InitOptions, error) {
	var level *int
	if text := strings.TrimSpace(f.level.Value()); text != "" && slices.Contains(f.visible(), initLevel) {
		value, err := strconv.Atoi(text)
		if err != nil {
			return repository.InitOptions{}, fmt.Errorf("%w %q: it must be a whole number", repository.ErrInvalidCompressionLevel, text)
		}
		level = &value
	}

	publicKey := f.publicKey
	if f.encryptionName() != encryptionConfigured {
		publicKey = nil
	}

	options := repository.NewInitOptions(f.compressionName(), level, publicKey).WithFormatVersion(f.format)

	return options, options.Validate()
}

func (f *initForm) submit() (bool, tea.Cmd) {
	options, err := f.options()
	if err != nil {
		f.err = err
		return false, nil
	}

	if f.encryptionName() != encryptionNewKeys {
		return true, emit(repositoryInitConfirmed{options: options})
	}

	if _, err := describeKeysFolder(f.keysFolder.Value()); err != nil {
		f.err = err
		return false, nil
	}
	folder, err := resolveLocalPath(f.keysFolder.Value())
	if err != nil {
		f.err = err
		return false, nil
	}

	return true, emit(encryptedInitFolderChosen{folder: folder, options: options})
}

func (f *initForm) view() string {
	var builder strings.Builder

	builder.WriteString(headerStyle.Render(f.title) + "\n\n")
	builder.WriteString("No Stowmark repository found at:\n" + highlightStyle.Render(f.location) + "\n\n")
	builder.WriteString(mutedStyle.Render("Prefilled with the configuration. Compression and format cannot change later.") + "\n\n")

	for _, field := range f.visible() {
		builder.WriteString(f.row(field) + "\n")
	}

	if note := f.note(); note != "" {
		builder.WriteString("\n" + mutedStyle.Render(note) + "\n")
	}
	if f.warning != "" {
		builder.WriteString("\n" + errorStyle.Render(f.warning) + "\n")
	}
	if f.err != nil {
		builder.WriteString("\n" + errorStyle.Render(f.err.Error()) + "\n")
	}

	builder.WriteString("\n[Tab/↑↓] Field  [←→] Change  [Enter] Initialize  [Esc] " + f.declineHint)

	return builder.String()
}

func (f *initForm) row(field initField) string {
	labels := map[initField]string{
		initCompression: "Compression",
		initLevel:       "Level",
		initFormat:      "Format",
		initEncryption:  "Encryption",
		initKeysFolder:  "Keys folder",
	}

	label := fmt.Sprintf("%-*s", initLabelWidth, labels[field])
	if field == f.focus {
		label = highlightStyle.Render(label)
	}

	switch field {
	case initLevel:
		levels, _ := repository.LevelsOf(f.compressionName())
		return label + inputStyle.Width(initLevelWidth+3).Render(f.level.View()) + "  " +
			mutedStyle.Render(fmt.Sprintf("%d to %d, empty for %d", levels.Minimum, levels.Maximum, levels.Default))
	case initKeysFolder:
		return label + inputStyle.Width(initFolderWidth+3).Render(f.keysFolder.View())
	case initCompression:
		return label + f.selector(field, f.compressionName()) + "  " + mutedStyle.Render(strings.Join(repository.Compressions(), ", "))
	case initFormat:
		return label + f.selector(field, "version "+strconv.Itoa(f.format))
	case initEncryption:
		value := f.encryptionName()
		if value == encryptionConfigured {
			value += " " + *f.publicKey
		}
		return label + f.selector(field, value)
	}

	return label
}

func (f *initForm) selector(field initField, value string) string {
	if field != f.focus || len(f.choices(field)) < 2 {
		return "  " + value
	}

	return highlightStyle.Render("◂ " + value + " ▸")
}

func (f *initForm) choices(field initField) []string {
	switch field {
	case initCompression:
		return repository.Compressions()
	case initEncryption:
		return f.encryptions
	case initFormat:
		return []string{"1", "2", "3"}
	case initLevel, initKeysFolder:
	}

	return nil
}

func (f *initForm) note() string {
	encrypted := f.encryptionName() != encryptionNone
	switch {
	case encrypted && f.format < repository.LatestFormatVersion:
		return "Format version 3 is recommended for encrypted repositories: it also encrypts the manifests."
	case f.encryptionName() == encryptionNewKeys:
		return "A new key pair is generated in the folder and this session opens the repository with it."
	case !encrypted:
		return "Without encryption anyone with access to the storage can read the snapshots."
	}

	return ""
}
