// pinyin-demo 在终端里模拟一款拼音输入法的完整按键体验：
// 敲字母 → 实时候选 → 数字/空格选词 → 上屏，全程离线。
//
// 按键说明：
//   a-z / '   输入拼音（' 分隔音节）
//   Backspace 删除
//   1-9       选候选
//   空格      选第一个候选
//   Enter     原样上屏剩余字母并退出
//   Esc / C-c 退出
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"pinyinyx/internal/dict"
	"pinyinyx/internal/engine"

	"golang.org/x/term"
)

func main() {
	dictPath := flag.String("dict", "data/dict.txt", "主词典路径")
	userPath := flag.String("user", "data/user.txt", "用户词频路径")
	flag.Parse()

	store, err := dict.Load(*dictPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "加载词典失败:", err)
		os.Exit(1)
	}
	store.LoadUser(*userPath)
	eng := engine.New(store)

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Fprintln(os.Stderr, "终端进入原始模式失败（需在真实终端中运行）:", err)
		os.Exit(1)
	}
	defer term.Restore(fd, oldState)

	var (
		committed string // 已上屏文字
		pinyin    []byte // 当前拼音缓冲（仅 a-z 和 '）
		prevLines int
	)

	draw := func() {
		if prevLines > 0 {
			fmt.Printf("\x1b[%dA\x1b[J", prevLines)
		}
		input := string(pinyin)
		lines := 0
		fmt.Printf("\r已上屏: %s\x1b[33m%s\x1b[0m\x1b[K\n", committed, input)
		lines++
		fmt.Printf("\r拼音: %s\x1b[K\n", input)
		lines++
		if input != "" {
			var sb strings.Builder
			for i, c := range eng.Candidates(input, 9) {
				fmt.Fprintf(&sb, "%d.%s  ", i+1, c.Word)
			}
			fmt.Printf("\r\x1b[36m%s\x1b[0m\x1b[K\n", sb.String())
			lines++
		}
		prevLines = lines
	}

	pick := func(idx int) {
		cands := eng.Candidates(string(pinyin), 9)
		if idx >= len(cands) {
			return
		}
		ent := cands[idx]
		eng.Choose(ent)
		committed += ent.Word
		pinyin = pinyin[:0]
	}

	buf := make([]byte, 1)
	draw()
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			break
		}
		c := buf[0]
		switch {
		case c == 3 || c == 27: // C-c / Esc
			goto done
		case c == '\r' || c == '\n':
			committed += string(pinyin)
			goto done
		case c == 127 || c == 8: // Backspace
			if len(pinyin) > 0 {
				pinyin = pinyin[:len(pinyin)-1]
			}
		case c >= 'a' && c <= 'z' || c == '\'':
			pinyin = append(pinyin, c)
		case c == ' ':
			pick(0)
		case c >= '1' && c <= '9':
			pick(int(c - '1'))
		}
		draw()
	}
done:
	if prevLines > 0 {
		fmt.Printf("\x1b[%dA\x1b[J", prevLines)
	}
	fmt.Printf("\r最终上屏: %s\n", committed)
	if err := store.SaveUser(); err != nil {
		fmt.Fprintln(os.Stderr, "保存用户词频失败:", err)
	}
}
