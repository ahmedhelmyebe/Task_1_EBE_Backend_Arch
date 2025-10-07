// shows where to put logic that doesn’t depend on Gin/GORM—easy to unit test.
// Place for pure domain logic
package core

import "strings"

// Small, framework-agnostic logic demo.
// NormalizeName is a tiny example of "pure" core logic that doesn't depend on HTTP/DB frameworks.
// Keeping domain rules here makes it highly testable and reusable.
func NormalizeName(s string) string {
	s = strings.TrimSpace(s) // Remove leading/trailing whitespace (clean user input).
	if s == "" { //if empty after triming , return as 
		return s
	}
	//upercase first leyyer to standrize display 
	return strings.ToUpper(s[:1]) + s[1:]
}
