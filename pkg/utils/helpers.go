package utils

import "strings"

// Helper function to check if a string is empty
func IsEmpty(s string) bool {
	return len(s) == 0
}

// Helper function to concatenate two strings
func Concat(a, b string) string {
	return a + b
}

// Helper function to remove leading and trailing whitespace from a string
func Trim(s string) string {
	return strings.TrimSpace(s)
}
