package assets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// AppIcon returns the application icon
func AppIcon() fyne.Resource {
	return theme.FileTextIcon()
}
