package utils

import (
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func toTitle(str string) string {
	return cases.Title(language.Und, cases.NoLower).String(str)
}
