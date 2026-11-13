// Package tone 提供带声调拼音到无声调编码的归一化。
package tone

import "strings"

var toneMap = map[rune]rune{
	'ā': 'a', 'á': 'a', 'ǎ': 'a', 'à': 'a',
	'ē': 'e', 'é': 'e', 'ě': 'e', 'è': 'e',
	'ī': 'i', 'í': 'i', 'ǐ': 'i', 'ì': 'i',
	'ō': 'o', 'ó': 'o', 'ǒ': 'o', 'ò': 'o',
	'ū': 'u', 'ú': 'u', 'ǔ': 'u', 'ù': 'u',
	'ǖ': 'v', 'ǘ': 'v', 'ǚ': 'v', 'ǜ': 'v',
	'ü': 'v',
	'ń': 'n', 'ň': 'n', 'ǹ': 'n',
	'ê': 'e', 'ḿ': 'm',
}

// Normalize 将带声调拼音转为键盘编码：去声调、ü→v，其余转小写。
// 例: "zhōng" → "zhong", "lǜ" → "lv"。
func Normalize(py string) string {
	py = strings.ToLower(strings.TrimSpace(py))
	var b strings.Builder
	for _, r := range py {
		if c, ok := toneMap[r]; ok {
			b.WriteRune(c)
		} else if r >= 'a' && r <= 'z' {
			b.WriteRune(r)
		}
		// 其他字符（数字、标点等）丢弃
	}
	return b.String()
}
