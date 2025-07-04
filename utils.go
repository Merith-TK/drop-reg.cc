package main

import "regexp"

// Discord URL validation regex
var DiscordURLRegex = regexp.MustCompile(`^https://discord\.gg/[a-zA-Z0-9]+$`)
