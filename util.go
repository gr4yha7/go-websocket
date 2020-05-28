package main

import (
	"regexp"
	"strings"
	"unicode"
)

func WordCount(value string) int {
	// Match non-space character sequences.
	re := regexp.MustCompile(`[\S]+`)

	// Find all matches and return count.
	results := re.FindAllString(value, -1)
	return len(results)
}

func trimWhitespace(str string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, str)
}

func CharacterCount(word string) int {
	// trim whitespaces
	wordNoWhitespace := trimWhitespace(word)
	return len([]rune(wordNoWhitespace))
}
