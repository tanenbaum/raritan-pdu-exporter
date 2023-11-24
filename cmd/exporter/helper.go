package main

import (
	"fmt"
	"regexp"
	"strings"
)

func matchAnyFilter(a string, patternList []string) bool {
	for _, p := range patternList {
		reg := regexp.MustCompile("^" + strings.ReplaceAll(p, "*", ".*") + "$")
		if reg.MatchString(a) {
			fmt.Printf("%s does match %s\n", a, p)
			return true
		} else {
			fmt.Printf("%s does not match %s\n", a, p)
		}
	}
	return false
}

func listContains(lst []string, a string) bool {
	for _, p := range lst {
		if p == "all" {
			return true
		} else if p == a {
			return true
		}
	}
	return false
}
