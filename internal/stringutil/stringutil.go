package stringutil

// Truncate shortens a string to maxLen based on display width,
// appending "..." if truncated. Handles wide (CJK) characters.
func Truncate(s string, maxLen int) string {
	if RuneWidth(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	runes := []rune(s)
	w := 0
	for i, r := range runes {
		w += CharWidth(r)
		if w > maxLen-3 {
			return string(runes[:i]) + "..."
		}
	}
	return s
}

// TruncateSimple shortens a string by byte length, appending "..." if truncated.
func TruncateSimple(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// RuneWidth returns the display width of a string, accounting for wide characters.
func RuneWidth(s string) int {
	w := 0
	for _, r := range s {
		w += CharWidth(r)
	}
	return w
}

// CharWidth returns the display width of a rune (2 for CJK, 1 otherwise).
func CharWidth(r rune) int {
	if r >= 0x1100 &&
		(r <= 0x115f || r == 0x2329 || r == 0x232a ||
			(r >= 0x2e80 && r <= 0xa4cf && r != 0x303f) ||
			(r >= 0xac00 && r <= 0xd7a3) ||
			(r >= 0xf900 && r <= 0xfaff) ||
			(r >= 0xfe10 && r <= 0xfe19) ||
			(r >= 0xfe30 && r <= 0xfe6f) ||
			(r >= 0xff00 && r <= 0xff60) ||
			(r >= 0xffe0 && r <= 0xffe6) ||
			(r >= 0x20000 && r <= 0x2fffd) ||
			(r >= 0x30000 && r <= 0x3fffd)) {
		return 2
	}
	return 1
}
