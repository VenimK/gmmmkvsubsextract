package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// CustomIcon returns the application icon
func CustomIcon() fyne.Resource {
	// For now, use a built-in icon that represents subtitles
	return theme.FileTextIcon()
}
