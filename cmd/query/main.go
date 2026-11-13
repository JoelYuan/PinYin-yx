// query 供交叉核对用的候选查询：对给定拼音串打印前 n 个候选及得分依据。
// 用法: query ni'hao [-n 10] [-dict data/dict.txt] [-user data/user.txt]
package main

import (
	"flag"
	"fmt"
	"os"

	"pinyinyx/internal/dict"
	"pinyinyx/internal/engine"
)

func main() {
	dictPath := flag.String("dict", "data/dict.txt", "主词典路径")
	userPath := flag.String("user", "data/user.txt", "用户词频路径")
	n := flag.Int("n", 10, "候选数量")
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "用法: query <拼音串>")
		os.Exit(1)
	}

	store, err := dict.Load(*dictPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "加载词典失败:", err)
		os.Exit(1)
	}
	store.LoadUser(*userPath)
	eng := engine.New(store)

	for _, q := range flag.Args() {
		fmt.Printf("输入: %s\n", q)
		for i, c := range eng.Candidates(q, *n) {
			fmt.Printf("  %d. %-12s 拼音=%-20s 词频=%d 用户=%d\n",
				i+1, c.Word, fmt.Sprint(c.Syllables), c.Freq, store.UserFreq(c.Word))
		}
	}
}
