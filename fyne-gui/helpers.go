package main

// Helper function to check if a string matches any language name in the map
func containsLanguageName(text string, languages map[string]string) bool {
	for langName := range languages {
		if text == langName {
			return true
		}
	}
	return false
}
