package slogx

import (
	"strings"
	"unicode"
)

func SecureString(raw string, opts ...SecureStringOption) string {

	spec := newSecureStringOptions(opts)

	if n := len(raw) - int(spec.Suffix); n >= 0 {
		if n := n - int(spec.Prefix); n >= 0 {
			// form: "prefix:hidden:suffix"
			s := raw[0:spec.Prefix]
			s += strings.Repeat("#", int(spec.Count))
			s += raw[len(raw)-int(spec.Suffix):]
			return s
		}
		// form: "hidden:suffix"
		s := strings.Repeat("#", int(spec.Count))
		s += raw[len(raw)-int(spec.Suffix):]
		return s
	}
	// form: "hidden"
	s := strings.Repeat("#", int(spec.Count))
	return s
}

type SecureStringOptions struct {
	Rune   rune // hidden rune
	Count  uint // hidden part: repeate rune count
	Prefix uint // prefix part: show runes count
	Suffix uint // suffix part: show runes count
}

type SecureStringOption func(opts *SecureStringOptions)

func SecureStringRune(c rune) SecureStringOption {
	return func(opts *SecureStringOptions) {
		if !unicode.IsPrint(c) {
			return // ignore
		}
		opts.Rune = c
	}
}

func SecureStringRuneCount(n uint) SecureStringOption {
	return func(opts *SecureStringOptions) {
		opts.Count = max(n, 5)
	}
}

func SecureStringPrefix(n uint) SecureStringOption {
	return func(opts *SecureStringOptions) {
		opts.Prefix = n
	}
}

func SecureStringSuffix(n uint) SecureStringOption {
	return func(opts *SecureStringOptions) {
		opts.Suffix = n
	}
}

func newSecureStringOptions(opts []SecureStringOption) (spec SecureStringOptions) {
	spec = SecureStringOptions{
		Rune:   '#',
		Count:  8,
		Prefix: 8,
		Suffix: 4,
	}
	for _, option := range opts {
		option(&spec)
	}
	return spec
}
