package reference

const (
	EscapeCharacter = '\\'
)

type KMP struct {
	pattern string
	prefix  []int
	size    int
}

// compile new prefix-array given argument
func NewKMP(pattern string) *KMP {
	prefix := computePrefix(pattern)
	return &KMP{
		pattern: pattern,
		prefix:  prefix,
		size:    len(pattern)}
}

// returns an array containing indexes of matches
// - error if pattern argument is less than 1 char
func computePrefix(pattern string) []int {
	// sanity check
	len_p := len(pattern)
	if len_p < 2 {
		if len_p == 0 {
			panic("'pattern' must contain at least one character")
		}
		return []int{-1}
	}
	t := make([]int, len_p)
	t[0], t[1] = -1, 0

	pos, count := 2, 0
	for pos < len_p {

		if pattern[pos-1] == pattern[count] {
			count++
			t[pos] = count
			pos++
		} else {
			if count > 0 {
				count = t[count]
			} else {
				t[pos] = 0
				pos++
			}
		}
	}
	return t
}

// for effeciency, define default array size
const startSize = 10

// find every occurence of the kmp.pattern in 's'
func (kmp *KMP) FindAllStringIndex(s string) []int {
	// precompute
	len_s := len(s)

	if len_s < kmp.size {
		return []int{}
	}

	match := make([]int, 0, startSize)
	m, i := 0, 0
	for m+i < len_s {
		if kmp.pattern[i] == s[m+i] {
			if i == kmp.size-1 {
				// the word was matched
				match = append(match, m)
				// simulate miss, and keep running
				m = m + i - kmp.prefix[i]
				if kmp.prefix[i] > -1 {
					i = kmp.prefix[i]
				} else {
					i = 0
				}
			} else {
				i++
			}
		} else {
			m = m + i - kmp.prefix[i]
			if kmp.prefix[i] > -1 {
				i = kmp.prefix[i]
			} else {
				i = 0
			}
		}
	}
	return match
}
