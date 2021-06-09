package helpers

import (
	"strings"

	"github.com/google/logger"
	"gopkg.in/yaml.v2"
)

// Allowlist is a target struct we populate with values based on an allowlist yaml. To avoid a large list of duplicate entries
// we use both Globally allowed packages and Image specific packages we allow.
type Allowlist struct {
	GlobalPackages    []string            `yaml:"global_allowlist"`
	ComponentPackages map[string][]string `yaml:"container_allowlist"`
}

// CreateAllowList opens and parses an allowlistFile (yaml, see README.MD for format) and outputs and Allowlist object and
// an error. If no allowListFile is specified just returns an empty object to simplify downstream logic.
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

// FlattenAllowlist is used to combine entries from global part and container specific entries in the allowlist for easier
// checking for hits by InAllowList. Takes an Allowlist and containername. (container name is used here without baseRepo which is trimmed off)
// Returns a []string with allowed packages.
// TODO: Do we handle empty Allowlist.GlobalPackages?
func FlattenAllowlist(a *Allowlist, c string) (allowlist []string) {
	allowlist = a.GlobalPackages
	if a.ComponentPackages[c] != nil {
		allowlist = append(allowlist, a.ComponentPackages[c]...)
	}
	return allowlist
}

// InAllowList tests if a queried string is present in a list of strings.
// Returns a boolean and the matching item on the list (this is useful because we can query substrings)
func InAllowList(list []string, query string) (found bool, hit string) {
	for v := range list {
		// We check for the query using HasPrefix because this allows us to specify package with and without
		// a specific version (format in allow list is "package@version", the latter part being optional)
		if strings.HasPrefix(query, list[v]) {
			return true, list[v]
		}
	}
	return false, "none"
}

// allowlistParser is a helper function of CreateAllowlist that unmarshals a []byte into an Allowlist and then returns
// that allowlist and an error
func allowlistParser(data []byte) (Allowlist, error) {

	al := new(Allowlist)

	err := yaml.Unmarshal(data, al)

	return *al, err
}
