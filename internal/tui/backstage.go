package tui

import "github.com/charmbracelet/lipgloss"

const (
	clrPrimary   = "#7C3AED"
	clrAccent    = "#A78BFA"
	clrSuccess   = "#10B981"
	clrWarning   = "#F59E0B"
	clrError     = "#EF4444"
	clrMuted     = "#9CA3AF"
	clrText      = "#F3F4F6"
	clrBorder    = "#4B5563"
	clrHighlight = "#2D2460"
	clrSubtle    = "#6B7280"
	clrDim       = "#374151"
)

const logoASCII = "" +
	" ██╗   ██╗ ██████╗ ██╗  ██╗██╗███████╗    ███████╗ ██████╗ ██████╗ ████████╗\n" +
	" ████╗  ██║██╔═══██╗╚██╗██╔╝██║██╔════╝    ██╔════╝██╔═══██╗██╔══██╗╚══██╔══╝\n" +
	" ██╔██╗ ██║██║   ██║ ╚███╔╝ ██║█████╗      ███████╗██║   ██║██████╔╝   ██║   \n" +
	" ██║╚██╗██║██║   ██║ ██╔██╗ ██║██╔══╝      ╚════██║██║   ██║██╔══██╗   ██║   \n" +
	" ██║ ╚████║╚██████╔╝██╔╝ ██╗██║███████╗    ███████║╚██████╔╝██║  ██║   ██║   \n" +
	" ╚═╝  ╚═══╝ ╚═════╝ ╚═╝  ╚═╝╚═╝╚══════╝   ╚══════╝ ╚═════╝ ╚═╝  ╚═╝   ╚═╝  "

var (
	sLogoTuned = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrAccent)).
			Bold(true)

	sLogoPending = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrWarning)).
			Bold(true)

	sBadgeTuned = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color(clrSuccess)).
			Bold(true).
			Padding(0, 1)

	sBadgePending = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1F2937")).
			Background(lipgloss.Color(clrWarning)).
			Bold(true).
			Padding(0, 1)

	sPanelActive = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(clrPrimary)).
			PaddingLeft(2).PaddingRight(2).
			PaddingTop(1).PaddingBottom(1)

	sPanelInactive = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(clrBorder)).
			PaddingLeft(2).PaddingRight(2).
			PaddingTop(1).PaddingBottom(1)

	sPanelTitle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrPrimary)).
			Bold(true)

	sCursor = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrAccent)).
		Bold(true)

	sLabel = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrMuted)).
		Width(16)

	sValue = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrText))

	sValueEditing = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrPrimary)).
			Underline(true)

	sHighlightedRow = lipgloss.NewStyle().
			Background(lipgloss.Color(clrHighlight))

	sError = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrError))

	sSuccess = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrSuccess))

	sHintBar = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrSubtle)).
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(clrBorder)).
			MarginTop(1).
			PaddingTop(1)

	sKey = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrAccent)).
		Bold(true)

	sPickPanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(clrPrimary)).
			Padding(1, 3)

	sPickTitle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrText)).
			Bold(true).
			MarginBottom(1)

	sPickItemActive = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrAccent)).
			Bold(true)

	sPickItemNormal = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrMuted))

	sRuleExpanded = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrAccent)).
			Bold(true)

	sRuleCollapsed = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrText))

	sRuleMeta = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrSubtle))

	sRuleSubLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrMuted)).
			Width(14)
)

type fieldKind int

const (
	fkText fieldKind = iota
	fkTextList
	fkNumber
	fkToggle
)

type configField struct {
	id    int
	label string
	kind  fieldKind
	hint  string
}

type ruleEntry struct {
	name       string
	targetDir  string
	extensions string
	expanded   bool
	subCursor  int
}

const (
	fidScanDir    = 0
	fidScanDirs   = 1
	fidLogsDir    = 2
	fidMaxSize    = 3
	fidMaxAge     = 4
	fidMaxBackups = 5
	fidCompress   = 6
)

var allFields = []configField{
	{id: fidScanDir, label: "Scan Dir", kind: fkText, hint: "Primary directory to watch for new files"},
	{id: fidScanDirs, label: "Scan Dirs", kind: fkTextList, hint: "Additional directories (comma-separated)"},
	{id: fidLogsDir, label: "Logs Dir", kind: fkText, hint: "Directory where log files are written"},
	{id: fidMaxSize, label: "Max Log Size", kind: fkNumber, hint: "Max size in MB before log rotation"},
	{id: fidMaxAge, label: "Max Log Age", kind: fkNumber, hint: "Days to retain old log files"},
	{id: fidMaxBackups, label: "Log Backups", kind: fkNumber, hint: "Number of rotated backup files to keep"},
	{id: fidCompress, label: "Compress Logs", kind: fkToggle, hint: "Compress rotated log files with gzip"},
}
