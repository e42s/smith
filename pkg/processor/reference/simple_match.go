package reference

import (
	"strings"
	"bytes"
	"fmt"
	"errors"
)

type SimpleMatcher struct {
	prefix string
	suffix string
	prefixKMP *KMP
	suffixKMP *KMP
}

func NewSimpleMatcher(prefix string, suffix string) *SimpleMatcher {
	return &SimpleMatcher{
		prefix: prefix,
		suffix: suffix,
		prefixKMP: NewKMP(prefix),
		suffixKMP: NewKMP(prefix),
	}
}

func (m *SimpleMatcher) MatchString(s string) bool {
	return strings.HasPrefix(s, m.prefix) && strings.HasSuffix(s, m.suffix)
}

func (m *SimpleMatcher) ReplaceAllStringFunc(src string, repl func(string) string) (string, error) {
	var buffer bytes.Buffer

	prefixIndices := m.prefixKMP.FindAllStringIndex(src)
	suffixIndices := m.suffixKMP.FindAllStringIndex(src)

	if len(prefixIndices) == 0 || len(suffixIndices) == 0 {
		return src, nil
	}
	if len(prefixIndices) != len(suffixIndices) {
		return nil, errors.New("prefix and suffix counts don't match")
	}

	// TODO: nested references are not supported by this algorithm. do we need nested support?
	for i, start := range prefixIndices {
		start += len(m.prefix) // skip prefix
		end := suffixIndices[i] // skip suffix
		buffer.WriteString(repl(src[start:end]))
	}

	for pos, char := range src {
		fmt.Printf("character %c starts at byte position %d\n", char, pos)
	}

	return fmt.Sprint(buffer.String()), nil
}
