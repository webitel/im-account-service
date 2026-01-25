package model

import (
	"regexp"
	"strings"
)

type ContactName struct {
	// End-User's full name in displayable form including all name parts,
	// possibly including titles and suffixes, ordered according to the End-User's locale and preferences.
	CommonName string // FullName
	// Given name(s) or first name(s) of the End-User.
	// Note that in some cultures, people can have multiple given names;
	// all can be present, with the names being separated by space characters.
	GivenName string // FirstName
	// Middle name(s) of the End-User.
	// Note that in some cultures, people can have multiple middle names;
	// all can be present, with the names being separated by space characters.
	// Also note that in some cultures, middle names are not used.
	MiddleName string // SecondName
	// Surname(s) or last name(s) of the End-User.
	// Note that in some cultures, people can have multiple family names or no family name;
	// all can be present, with the names being separated by space characters.
	FamilyName string // LastName
}

// [ Given | Middle | Family ] non-empty parts
func (cn *ContactName) Parts() []string {
	if cn == nil {
		return nil
	}
	parts := []string{
		cn.GivenName,
		cn.MiddleName,
		cn.FamilyName,
	}
	for e := 0; e < len(parts); e++ {
		// trim leading / trailing WSP..
		part := strings.TrimSpace(parts[e])
		// replace WSP+ sequence with a single SPACE
		part = regexpWSPlus.ReplaceAllString(part, string(SPACE))
		if part != "" {
			parts[e] = part // normalized
			continue
		}
		parts = append(parts[:e], parts[e+1:]...)
		e-- // removed
	}
	return parts
}

func (cn ContactName) LastName() string {
	parts := cn.Parts()
	if n := len(parts); n > 1 {
		return parts[n-1]
	}
	return ""
}

func (cn ContactName) FirstName() string {
	parts := cn.Parts()
	if n := len(parts); n > 0 {
		return CommonName(parts[:n-1]...)
	}
	return cn.CommonName
}

func (cn *ContactName) IsValid() bool {
	if cn == nil {
		return false
	}
	if cn.CommonName != "" {
		return true
	}
	return len(cn.Parts()) > 0
}

// regular expression that matches one or more whitespace characters (\s+).
// \s represents any whitespace character (space, tab, newline, etc.)
// + means "one or more" occurrences
var regexpWSPlus = regexp.MustCompile(`\s+`)

// CommonName forms full name form given part(s)
func CommonName(parts ...string) string {
	var form strings.Builder
	defer form.Reset()
	const SPACE = string(SPACE)
	var sep string // separator
	for _, part := range parts {
		part = strings.TrimSpace(part)
		// replace WSP sequence with a single SP
		part = regexpWSPlus.ReplaceAllString(
			part, SPACE,
		)
		if part == "" {
			continue
		}
		form.WriteString(sep)
		form.WriteString(part)
		sep = SPACE
	}
	// generated
	return form.String()
}

// String returns .CommonName form
func (cn *ContactName) String() string {
	if cn.CommonName == "" {
		// generated
		cn.CommonName = CommonName(
			cn.GivenName,  // firstName
			cn.MiddleName, // secondName
			cn.FamilyName, // lastName
		)
	}
	return cn.CommonName
}
