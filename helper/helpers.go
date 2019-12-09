package helpers

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strings"
)

func Check(e error, message string) {
	if e != nil {
		fmt.Printf("%s \n %e", message, e)
		panic(e)
	}
}
func CompositionParser(compositionFile string) (map[string]string, error) {
	zdComposition := make(map[string]string)
	containerList := make(map[string]string)
	yamlFile, err := ioutil.ReadFile(compositionFile)
	Check(err, fmt.Sprintf("Failed to load %s, #%e", compositionFile, err))

	err = yaml.Unmarshal(yamlFile, zdComposition)
	Check(err, fmt.Sprintf("Failed to unmarshal %v, #%e", yamlFile, err))

	for c, v := range zdComposition {
		c = underscoreHyphenator(versionStripper(c))
		containerList[c] = v
	}

	return containerList, err
}

func versionStripper(input string) (output string) {
	return strings.Replace(input, "_version", "", 1)
}

func underscoreHyphenator(input string) (output string) {
	return strings.Replace(input, "_", "-", -1)
}
