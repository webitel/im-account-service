package model

const (
	SPACE  byte = ' '
	ESCAPE byte = '\\'
)

func IsAlpha(c rune) bool {
	return IsLower(c) || IsUpper(c)
}

func IsLower(c rune) bool {
	return ('a' <= c && c <= 'z')
}

func IsUpper(c rune) bool {
	return ('A' <= c && c <= 'Z')
}

func IsDigit(c rune) bool {
	return ('0' <= c && c <= '9')
}
