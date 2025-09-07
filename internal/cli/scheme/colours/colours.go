package colours

import "github.com/fatih/color"

// Color scheme for the CLI
var (
	Title   = color.New(color.FgCyan, color.Bold)
	Author  = color.New(color.FgMagenta)
	Prompt  = color.New(color.FgGreen, color.Bold)
	Error   = color.New(color.FgRed, color.Bold)
	Success = color.New(color.FgGreen)
	Info    = color.New(color.FgBlue)
	Warning = color.New(color.FgYellow)
)
