package handlers

import "regexp"

var feedRegex = regexp.MustCompile(`^\w+(\w-)*\w+$`)
