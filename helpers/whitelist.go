package helpers

import (
	"strings"

	"github.com/google/logger"
	"gopkg.in/yaml.v2"
)

type Whitelist struct {
	GlobalPackages    []string            `yaml:"global_whitelist"`
	ComponentPackages map[string][]string `yaml:"container_whitelist"`
}

func CreateWhitelist(whitelistFile string, l logger.Logger) (whitelist Whitelist, err error) {
	if whitelistFile != "" {
		var wlBytes []byte
		wlBytes, err = fileReader(whitelistFile, &l)
		whitelist, err = whitelistParser(wlBytes)
	} else {
		whitelist = Whitelist{
			GlobalPackages:    []string{},
			ComponentPackages: map[string][]string{},
		}
	}
	return whitelist, err
}

func FlattenWhitelist(w *Whitelist, c string) (whitelist []string) {
	whitelist = w.GlobalPackages
	if w.ComponentPackages[c] != nil {
		whitelist = append(whitelist, w.ComponentPackages[c]...)
	}
	return whitelist
}

func InWhiteList(list []string, query string) (found bool, hit string) {
	for v := range list {
		if strings.HasPrefix(query, list[v]) {
			return true, list[v]
		}
	}
	return false, "none"
}

func whitelistParser(data []byte) (Whitelist, error) {

	wl := new(Whitelist)

	err := yaml.Unmarshal(data, wl)

	return *wl, err
}
