package parser

import "strings"

// SplitN and unpack a string
func SplitUnpack(str string, separator string) (string, string) {
	parsedStr := strings.SplitN(str, separator, 2)
	return parsedStr[0], parsedStr[1]
}

// Trim whitespace character such as line breaks or double spaces
func TrimWhiteSpaces(str string) string {
	trimWhiteSpace := strings.NewReplacer("\n", "", "  ", "")
	return trimWhiteSpace.Replace(str)
}
