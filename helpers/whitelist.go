package helpers

import (
	"github.com/google/logger"
	"gopkg.in/yaml.v2"
	"strings"
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

func whitelistParser(data []byte) (Whitelist, error) {

	wl := new(Whitelist)

	err := yaml.Unmarshal(data, wl)

	return *wl, err
}

func InWhiteList(list []string, query string) (found bool, hit string) {
	for v := range list {
		if strings.HasPrefix(query, list[v]) {
			return true, list[v]
		}
	}
	return false, "none"
}
