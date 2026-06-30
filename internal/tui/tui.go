package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Nox1KCL/Noxie-Sort/internal/config"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type screenID int

const (
	screenPickPath screenID = iota
	screenEditor
)

const (
	minWidth  = 96
	minHeight = 39
)

type focusArea int

const (
	focusFields focusArea = iota
	focusRules
)

type AppModel struct {
	width  int
	height int

	screen screenID

	pathOptions []string
	pathCursor  int

	configPath string
	cfg        *config.Config

	buf [7]string

	compress bool

	rules []ruleEntry

	focus       focusArea
	fieldCursor int
	rulesCursor int

	editing    bool
	editBuffer string

	editingRuleSub bool
	ruleBuf        string

	isTuned        bool
	validationErrs []string
	saveMsg        string
	saved          bool
}

func (m AppModel) Init() tea.Cmd {
	return nil
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch m.screen {
		case screenPickPath:
			return m.updatePickPath(msg)
		case screenEditor:
			return m.updateEditor(msg)
		}
	}

	return m, nil
}

func (m AppModel) updatePickPath(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "up", "k":
		if m.pathCursor > 0 {
			m.pathCursor--
		}

	case "down", "j":
		if m.pathCursor < len(m.pathOptions)-1 {
			m.pathCursor++
		}

	case "enter":
		chosen := m.pathOptions[m.pathCursor]
		if err := os.MkdirAll(filepath.Dir(chosen), 0o755); err != nil {
			m.saveMsg = fmt.Sprintf("Error creating directory: %v", err)
			m.saved = false
			return m, nil
		}

		defaultCfg, err := config.GetConfig("")
		if err != nil {
			m.saveMsg = fmt.Sprintf("Error loading default config: %v", err)
			m.saved = false
			return m, nil
		}
		data, err := marshalTOML(defaultCfg)
		if err != nil {
			m.saveMsg = fmt.Sprintf("Error serialising config: %v", err)
			m.saved = false
			return m, nil
		}
		if err := os.WriteFile(chosen, data, 0o644); err != nil {
			m.saveMsg = fmt.Sprintf("Error writing config: %v", err)
			m.saved = false
			return m, nil
		}
		m.configPath = chosen
		m.cfg = defaultCfg
		m.loadEditorFromConfig()
		m.screen = screenEditor
	}

	return m, nil
}

func (m AppModel) updateEditor(msg tea.KeyMsg) (tea.Model, tea.Cmd) {

	if !m.editing && !m.editingRuleSub {
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "ctrl+s":
			m.doSave()
			return m, nil
		}
	}

	if m.focus == focusFields {
		return m.updateFields(msg)
	}
	return m.updateRules(msg)
}

func (m AppModel) updateFields(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	field := allFields[m.fieldCursor]

	if m.editing {
		switch msg.String() {
		case "enter", "esc":
			m.buf[field.id] = m.editBuffer
			m.editing = false
			m.editBuffer = ""
			m.refreshTuned()
		case "backspace":
			if len(m.editBuffer) > 0 {
				m.editBuffer = m.editBuffer[:len(m.editBuffer)-1]
			}
		default:
			if len(msg.Runes) > 0 {
				m.editBuffer += string(msg.Runes)
			}
		}
		return m, nil
	}

	switch msg.String() {
	case "q":
		return m, tea.Quit

	case "up", "k":
		if m.fieldCursor > 0 {
			m.fieldCursor--
		}

	case "down", "j":
		if m.fieldCursor < len(allFields)-1 {
			m.fieldCursor++
		} else {

			m.focus = focusRules
		}

	case "tab":

		m.focus = focusRules

	case "enter", " ":
		switch field.kind {
		case fkToggle:
			m.compress = !m.compress
			m.refreshTuned()
		case fkText, fkTextList, fkNumber:
			m.editBuffer = m.buf[field.id]
			m.editing = true
		}
	}

	return m, nil
}

func (m AppModel) updateRules(msg tea.KeyMsg) (tea.Model, tea.Cmd) {

	if m.editingRuleSub {
		switch msg.String() {
		case "enter", "esc":
			if len(m.rules) > 0 {
				r := &m.rules[m.rulesCursor]
				switch r.subCursor {
				case 0:
					r.name = m.ruleBuf
				case 1:
					r.targetDir = m.ruleBuf
				case 2:
					r.extensions = m.ruleBuf
				}
			}
			m.editingRuleSub = false
			m.ruleBuf = ""
			m.refreshTuned()
		case "backspace":
			if len(m.ruleBuf) > 0 {
				m.ruleBuf = m.ruleBuf[:len(m.ruleBuf)-1]
			}
		default:
			if len(msg.Runes) > 0 {
				m.ruleBuf += string(msg.Runes)
			}
		}
		return m, nil
	}

	if len(m.rules) > 0 && m.rules[m.rulesCursor].expanded {
		r := &m.rules[m.rulesCursor]
		switch msg.String() {
		case "q":
			return m, tea.Quit

		case "ctrl+s":
			m.doSave()
			return m, nil

		case "up", "k":
			if r.subCursor > 0 {
				r.subCursor--
			} else {

				r.expanded = false
				if m.rulesCursor > 0 {
					m.rulesCursor--
				} else {
					m.focus = focusFields
				}
			}

		case "down", "j":
			if r.subCursor < 2 {
				r.subCursor++
			} else {

				r.expanded = false
				if m.rulesCursor < len(m.rules)-1 {
					m.rulesCursor++
				}
			}

		case "esc":
			r.expanded = false

		case "enter", " ":

			switch r.subCursor {
			case 0:
				m.ruleBuf = r.name
			case 1:
				m.ruleBuf = r.targetDir
			case 2:
				m.ruleBuf = r.extensions
			}
			m.editingRuleSub = true

		case "a":
			m.addRule()

		case "d":
			m.deleteCurrentRule()
		}
		return m, nil
	}

	switch msg.String() {
	case "q":
		return m, tea.Quit

	case "ctrl+s":
		m.doSave()
		return m, nil

	case "up", "k":
		if m.rulesCursor > 0 {
			m.rulesCursor--
		} else {

			m.focus = focusFields
			m.fieldCursor = len(allFields) - 1
		}

	case "down", "j":
		if m.rulesCursor < len(m.rules)-1 {
			m.rulesCursor++
		}

	case "tab":

		m.focus = focusFields

	case "enter":
		if len(m.rules) > 0 {
			m.rules[m.rulesCursor].expanded = !m.rules[m.rulesCursor].expanded
			m.rules[m.rulesCursor].subCursor = 0
		}

	case "a":
		m.addRule()

	case "d":
		m.deleteCurrentRule()
	}

	return m, nil
}

func (m *AppModel) addRule() {
	m.rules = append(m.rules, ruleEntry{
		name:      fmt.Sprintf("Rule%d", len(m.rules)+1),
		targetDir: "",
		expanded:  true,
		subCursor: 0,
	})
	m.rulesCursor = len(m.rules) - 1
}

func (m *AppModel) deleteCurrentRule() {
	if len(m.rules) == 0 {
		return
	}
	m.rules = append(m.rules[:m.rulesCursor], m.rules[m.rulesCursor+1:]...)
	if m.rulesCursor >= len(m.rules) && m.rulesCursor > 0 {
		m.rulesCursor--
	}
	m.refreshTuned()
}

func (m *AppModel) doSave() {
	cfg, errs := m.buildConfig()
	if len(errs) > 0 {
		m.validationErrs = errs
		m.saveMsg = "Fix validation errors before saving."
		m.saved = false
		return
	}

	data, err := marshalTOML(cfg)
	if err != nil {
		m.saveMsg = fmt.Sprintf("Marshal error: %v", err)
		m.saved = false
		return
	}

	if err := os.MkdirAll(filepath.Dir(m.configPath), 0o755); err != nil {
		m.saveMsg = fmt.Sprintf("Cannot create directory: %v", err)
		m.saved = false
		return
	}
	if err := os.WriteFile(m.configPath, data, 0o644); err != nil {
		m.saveMsg = fmt.Sprintf("Write error: %v", err)
		m.saved = false
		return
	}

	m.cfg = cfg
	m.validationErrs = nil
	m.isTuned = true
	m.saveMsg = "Config saved successfully!"
	m.saved = true
}

func (m *AppModel) buildConfig() (*config.Config, []string) {
	var errs []string
	cfg := &config.Config{}

	cfg.ScanDir = strings.TrimSpace(m.buf[fidScanDir])

	for _, s := range strings.Split(m.buf[fidScanDirs], ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			cfg.ScanDirs = append(cfg.ScanDirs, s)
		}
	}

	if cfg.ScanDir == "" && len(cfg.ScanDirs) == 0 {
		errs = append(errs, "Scan Dir or Scan Dirs must not be empty")
	}

	cfg.LogsDir = strings.TrimSpace(m.buf[fidLogsDir])
	if cfg.LogsDir == "" {
		errs = append(errs, "Logs Dir must not be empty")
	}

	maxSize, err := strconv.Atoi(strings.TrimSpace(m.buf[fidMaxSize]))
	if err != nil || maxSize <= 0 {
		errs = append(errs, "Max Log Size must be a positive integer")
	} else {
		cfg.Logger.MaxSize = maxSize
	}

	maxAge, err := strconv.Atoi(strings.TrimSpace(m.buf[fidMaxAge]))
	if err != nil || maxAge <= 0 {
		errs = append(errs, "Max Log Age must be a positive integer")
	} else {
		cfg.Logger.MaxAge = maxAge
	}

	maxBackups, err := strconv.Atoi(strings.TrimSpace(m.buf[fidMaxBackups]))
	if err != nil || maxBackups < 0 {
		errs = append(errs, "Log Backups must be zero or a positive integer")
	} else {
		cfg.Logger.MaxBackups = maxBackups
	}

	cfg.Logger.Compress = m.compress

	if len(m.rules) == 0 {
		errs = append(errs, "At least one sorting rule is required")
	}

	cfg.Rules = make(map[string]config.FolderRule)
	for i, r := range m.rules {
		rName := strings.TrimSpace(r.name)
		rTarget := strings.TrimSpace(r.targetDir)
		if rName == "" {
			errs = append(errs, fmt.Sprintf("Rule #%d: name must not be empty", i+1))
			continue
		}
		if rTarget == "" {
			errs = append(errs, fmt.Sprintf("Rule %q: target dir must not be empty", rName))
		}
		var exts []string
		for _, e := range strings.Split(r.extensions, ",") {
			e = strings.TrimSpace(strings.ToLower(e))
			if e != "" {
				exts = append(exts, e)
			}
		}
		if len(exts) == 0 {
			errs = append(errs, fmt.Sprintf("Rule %q: at least one extension is required", rName))
		}
		cfg.Rules[rName] = config.FolderRule{
			TargetDir:  rTarget,
			Extensions: exts,
		}
	}

	if len(errs) == 0 {
		if vErr := cfg.ConfigExtValidate(); vErr != nil {
			for _, line := range strings.Split(vErr.Error(), "\n") {
				if line != "" {
					errs = append(errs, line)
				}
			}
		}
	}

	if len(errs) > 0 {
		return nil, errs
	}
	return cfg, nil
}

func (m *AppModel) loadEditorFromConfig() {
	if m.cfg == nil {
		return
	}
	m.buf[fidScanDir] = m.cfg.ScanDir
	m.buf[fidScanDirs] = strings.Join(m.cfg.ScanDirs, ", ")
	m.buf[fidLogsDir] = m.cfg.LogsDir
	m.buf[fidMaxSize] = strconv.Itoa(m.cfg.Logger.MaxSize)
	m.buf[fidMaxAge] = strconv.Itoa(m.cfg.Logger.MaxAge)
	m.buf[fidMaxBackups] = strconv.Itoa(m.cfg.Logger.MaxBackups)
	m.compress = m.cfg.Logger.Compress

	m.rules = make([]ruleEntry, 0, len(m.cfg.Rules))
	for name, rule := range m.cfg.Rules {
		m.rules = append(m.rules, ruleEntry{
			name:       name,
			targetDir:  rule.TargetDir,
			extensions: strings.Join(rule.Extensions, ", "),
		})
	}
	m.refreshTuned()
}

func (m *AppModel) refreshTuned() {
	_, errs := m.buildConfig()
	m.isTuned = len(errs) == 0
	if m.isTuned {
		m.validationErrs = nil
	}
}

func (m AppModel) View() string {
	if m.width > 0 && m.height > 0 && (m.width < minWidth || m.height < minHeight) {
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrWarning)).
				Bold(true).
				Render(fmt.Sprintf(
					"Terminal too small\n%d×%d required, got %d×%d",
					minWidth, minHeight, m.width, m.height,
				)),
		)
	}
	switch m.screen {
	case screenPickPath:
		return m.viewPickPath()
	case screenEditor:
		return m.viewEditor()
	}
	return ""
}

func (m AppModel) viewPickPath() string {
	title := sPickTitle.Render("No config file found.\nWhere would you like to create one?")

	var rows strings.Builder
	for i, p := range m.pathOptions {
		if i == m.pathCursor {
			rows.WriteString(sPickItemActive.Render("▶ " + p))
		} else {
			rows.WriteString(sPickItemNormal.Render("  " + p))
		}
		rows.WriteString("\n")
	}

	hints := "\n" + sKey.Render("↑↓/jk") + " navigate   " +
		sKey.Render("Enter") + " create & open   " +
		sKey.Render("q") + " quit"

	content := title + "\n\n" + rows.String() + hints
	panel := sPickPanel.Render(content)

	if m.width > 0 {
		panel = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, panel)
	}
	return panel
}

func (m AppModel) viewEditor() string {

	header := m.viewHeader()

	fieldsPanel := m.viewFieldsPanel()
	rulesPanel := m.viewRulesPanel()

	panelWidth := m.width - 4
	if panelWidth < 40 {
		panelWidth = 40
	}

	fieldsPanel = sPanelInactive.Width(panelWidth).Render(fieldsPanel)
	rulesPanel = sPanelInactive.Width(panelWidth).Render(rulesPanel)

	if m.focus == focusFields {
		fieldsPanel = sPanelActive.Width(panelWidth).Render(m.viewFieldsPanel())
	} else {
		rulesPanel = sPanelActive.Width(panelWidth).Render(m.viewRulesPanel())
	}

	statusLine := m.viewStatusLine()

	hints := m.viewHints()

	return header + "\n" + fieldsPanel + "\n" + rulesPanel + "\n" + statusLine + hints
}

func (m AppModel) viewHeader() string {
	var logo string
	var badge string

	if m.isTuned {
		logo = sLogoTuned.Render(logoASCII)
		badge = sBadgeTuned.Render(
			"╔══════════════╗\n" +
				"║  ◉  TUNED   ║\n" +
				"╚══════════════╝",
		)
	} else {
		logo = sLogoPending.Render(logoASCII)
		badge = sBadgePending.Render(
			"╔══════════════╗\n" +
				"║ ◌  PENDING  ║\n" +
				"╚══════════════╝",
		)
	}

	badgeWithPad := lipgloss.NewStyle().
		MarginTop(3).
		Render(badge)

	var logoLine string
	if m.width > 0 {
		logoLine = lipgloss.JoinHorizontal(
			lipgloss.Bottom,
			logo,
			lipgloss.NewStyle().Width(m.width-lipgloss.Width(logo)-lipgloss.Width(badge)-2).Render(""),
			badgeWithPad,
		)
	} else {
		logoLine = logo + "  " + badge
	}

	pathLine := lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrSubtle)).
		Italic(true).
		Render("  config: " + m.configPath)

	return logoLine + "\n" + pathLine + "\n"
}

func (m AppModel) viewFieldsPanel() string {
	title := sPanelTitle.Render("⚙  Basic Configuration")
	var sb strings.Builder
	sb.WriteString(title + "\n\n")

	for i, f := range allFields {
		isActive := m.focus == focusFields && i == m.fieldCursor
		isEditing := isActive && m.editing

		cursor := "  "
		if isActive {
			cursor = sCursor.Render("▶ ")
		}

		label := sLabel.Render(f.label)

		var value string
		switch {
		case isEditing:
			value = sValueEditing.Render(m.editBuffer + "█")
		case f.kind == fkToggle:
			if m.compress {
				value = sSuccess.Render("✓ enabled")
			} else {
				value = lipgloss.NewStyle().Foreground(lipgloss.Color(clrMuted)).Render("✗ disabled")
			}
		default:
			v := m.buf[f.id]
			if v == "" {
				value = lipgloss.NewStyle().Foreground(lipgloss.Color(clrSubtle)).Italic(true).Render("(empty)")
			} else {
				value = sValue.Render(v)
			}
		}

		row := cursor + label + value

		if isActive && !isEditing {

			row = sHighlightedRow.Render(row)
		}

		sb.WriteString(row + "\n")
	}

	return sb.String()
}

func (m AppModel) viewRulesPanel() string {
	title := sPanelTitle.Render("📁 Sorting Rules")
	var sb strings.Builder
	sb.WriteString(title + "\n\n")

	if len(m.rules) == 0 {
		sb.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrSubtle)).
			Italic(true).
			Render("  No rules defined. Press [a] to add one.") + "\n")
	}

	for i, r := range m.rules {
		isActive := m.focus == focusRules && i == m.rulesCursor
		cursor := "  "
		if isActive {
			cursor = sCursor.Render("▶ ")
		}

		extPreview := sRuleMeta.Render(r.extensions)
		if r.extensions == "" {
			extPreview = lipgloss.NewStyle().Foreground(lipgloss.Color(clrSubtle)).Italic(true).Render("(no extensions)")
		}

		if r.expanded {

			header := cursor + sRuleExpanded.Render("▼ "+r.name) + "\n"
			sb.WriteString(header)

			subFields := []struct {
				label string
				value string
				idx   int
			}{
				{"Name", r.name, 0},
				{"Target Dir", r.targetDir, 1},
				{"Extensions", r.extensions, 2},
			}

			for _, sf := range subFields {
				subActive := isActive && r.subCursor == sf.idx
				subEditing := subActive && m.editingRuleSub

				subCursor := "    "
				if subActive {
					subCursor = "  " + sCursor.Render("▷ ")
				}

				subLabel := sRuleSubLabel.Render(sf.label)
				var subValue string

				switch {
				case subEditing:
					subValue = sValueEditing.Render(m.ruleBuf + "█")
				case sf.value == "":
					subValue = lipgloss.NewStyle().
						Foreground(lipgloss.Color(clrSubtle)).
						Italic(true).
						Render("(empty)")
				default:
					subValue = sValue.Render(sf.value)
				}

				row := subCursor + subLabel + subValue
				if subActive && !subEditing {
					row = sHighlightedRow.Render(row)
				}
				sb.WriteString(row + "\n")
			}

		} else {

			nameStr := sRuleCollapsed.Render("▶ " + r.name)
			arrow := sRuleMeta.Render(" → " + r.targetDir + "   ")
			row := cursor + nameStr + arrow + extPreview + "\n"

			if isActive {
				row = sHighlightedRow.Render(cursor+nameStr+arrow+extPreview) + "\n"
			}
			sb.WriteString(row)
		}
	}

	sb.WriteString("\n" +
		sKey.Render("[a]") + " add   " +
		sKey.Render("[d]") + " delete   " +
		sKey.Render("[Enter]") + " expand/edit rule\n")

	return sb.String()
}

func (m AppModel) viewStatusLine() string {
	if m.saveMsg != "" {
		if m.saved {
			return sSuccess.Render("✔  "+m.saveMsg) + "\n"
		}
		return sError.Render("✖  "+m.saveMsg) + "\n"
	}

	if len(m.validationErrs) > 0 {
		var sb strings.Builder
		for _, e := range m.validationErrs {
			sb.WriteString(sError.Render("  ! "+e) + "\n")
		}
		return sb.String()
	}

	if m.focus == focusFields && !m.editing {
		hint := allFields[m.fieldCursor].hint
		if hint != "" {
			return lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrSubtle)).
				Italic(true).
				Render("  ⓘ  "+hint) + "\n"
		}
	}
	return ""
}

func (m AppModel) viewHints() string {
	var parts []string

	if m.editing || m.editingRuleSub {
		parts = []string{
			sKey.Render("Enter") + " confirm",
			sKey.Render("Esc") + " cancel",
			sKey.Render("Backspace") + " delete",
		}
	} else {
		parts = []string{
			sKey.Render("↑↓/jk") + " navigate",
			sKey.Render("Tab") + " switch panel",
			sKey.Render("Enter") + " edit / toggle",
			sKey.Render("Ctrl+S") + " save",
			sKey.Render("q") + " quit",
		}
	}

	return sHintBar.Width(m.width - 2).Render(strings.Join(parts, "   "))
}

func NewAppModel(configPath string, cfg *config.Config, paths []string) AppModel {
	m := AppModel{
		pathOptions: paths,
		pathCursor:  0,
	}

	if configPath == "" {
		m.screen = screenPickPath
	} else {
		m.configPath = configPath
		m.cfg = cfg
		m.screen = screenEditor
		m.loadEditorFromConfig()
	}

	return m
}
