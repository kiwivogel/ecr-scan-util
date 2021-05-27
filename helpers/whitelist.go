package helpers

import (
	"strings"

	"github.com/google/logger"
	"gopkg.in/yaml.v2"
)

type Allowlist struct {
	GlobalPackages    []string            `yaml:"global_allowlist"`
	ComponentPackages map[string][]string `yaml:"container_allowlist"`
}

func CreateAllowlist(allowListFile string, l logger.Logger) (allowlist Allowlist, err error) {
	if allowListFile != "" {
		var wlBytes []byte
		wlBytes, err = fileReader(allowListFile, &l)
		allowlist, err = allowlistParser(wlBytes)
	} else {
		allowlist = Allowlist{
			GlobalPackages:    []string{},
			ComponentPackages: map[string][]string{},
		}
	}
	return allowlist, err
}

func FlattenAllowlist(a *Allowlist, c string) (allowlist []string) {
	allowlist = a.GlobalPackages
	if a.ComponentPackages[c] != nil {
		allowlist = append(allowlist, a.ComponentPackages[c]...)
	}
	return allowlist
}

func InAllowList(list []string, query string) (found bool, hit string) {
	for v := range list {
		if strings.HasPrefix(query, list[v]) {
			return true, list[v]
		}
	}
	return false, "none"
}

func allowlistParser(data []byte) (Allowlist, error) {

	al := new(Allowlist)

	err := yaml.Unmarshal(data, al)

	return *al, err
}
