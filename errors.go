package main

import (
	"regexp"
	"strings"
)

var pattern = regexp.MustCompile(`([a-zA-Z0-9\-_]{41})`) // assuming base64url-encoded values to be 32 bytes long (43 chars) minus 2 chars prefix

func sanitizeError(err error) string {
	return pattern.ReplaceAllString(err.Error(), strings.Repeat("X", 41))
}
