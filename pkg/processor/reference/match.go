package reference

type Matcher interface {
	MatchString(s string) bool
	ReplaceAllStringFunc(src string, repl func(string) string) string
}
