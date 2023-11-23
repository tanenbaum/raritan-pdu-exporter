package exporter

import (
	"regexp"
	"strings"

	"github.com/iancoleman/strcase"
)

func snakeCase() func(string) string {
	ms := map[string]string{}

	return func(v string) string {
		t, ok := ms[v]
		if !ok {
			sc := strcase.ToSnake(v)
			ms[v] = sc
			return sc
		}
		return t
	}
}

func matchAnyFilter(a string, patternList []string) bool {
	for _, p := range patternList {
		reg := regexp.MustCompile("^" + strings.ReplaceAll(p, "*", ".*") + "$")
		if reg.MatchString(a) {
			return true
		} else if strings.ToLower(p) == "all" {
			return true
		}
	}
	return false
}
