package common

import "strings"

const (
	SecurityProfileNormal   = "normal"
	SecurityProfileBlackbox = "blackbox"
	SecurityProfileStrict   = "strict"
)

func NormalizeBlackboxLoginPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/client-login"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if len(path) > 1 {
		path = strings.TrimRight(path, "/")
	}
	return path
}

func IsBlackboxEnabled() bool {
	profile := strings.ToLower(strings.TrimSpace(SecurityProfile))
	return profile == SecurityProfileBlackbox || profile == SecurityProfileStrict
}

func IsBlackboxStrict() bool {
	return strings.EqualFold(strings.TrimSpace(SecurityProfile), SecurityProfileStrict)
}
