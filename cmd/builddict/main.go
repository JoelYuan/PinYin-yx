// builddict 将开源拼音数据合并为输入法主词典 data/dict.txt。
//
// 输入数据（均已随项目归档于 data/ 目录）：
//   - pinyin_data.txt  单字拼音   (mozillazg/pinyin-data, MIT)
//     格式: U+4E2D: zhōng,zhòng  # 中
//   - phrase_pinyin.txt 词组拼音 (mozillazg/phrase-pinyin-data, MIT)
//     格式: 爱国: ài guó
//   - jieba_dict.txt   简体词频  (fxsjy/jieba, MIT)
//     格式: 词 词频 词性
//
// 输出格式: 词<TAB>无声调拼音（空格分隔）<TAB>词频
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"pinyinyx/internal/tone"
)

const (
	cjkMin = 0x4E00
	cjkMax = 0x9FFF
)

func isHan(r rune) bool {
	return (r >= cjkMin && r <= cjkMax) || r == 0x3007
}

func main() {
	charIn := "data/pinyin_data.txt"
	phraseIn := "data/phrase_pinyin.txt"
	freqIn := "data/jieba_dict.txt"
	out := "data/dict.txt"
	for i, arg := range os.Args[1:] {
		switch i {
		case 0:
			charIn = arg
		case 1:
			phraseIn = arg
		case 2:
			freqIn = arg
		case 3:
			out = arg
		}
	}

	// 词频表
	freq := make(map[string]int, 400000)
	loadFreq(freqIn, freq)

	type key struct{ w, p string }
	seen := make(map[key]int, 600000)
	var order []key
	add := func(w, py string, f int) {
		k := key{w, py}
		if old, ok := seen[k]; ok {
			if f > old {
				seen[k] = f
			}
			return
		}
		seen[k] = f
		order = append(order, k)
	}

	// 1) 单字：多音字每个读音各收一条
	nChar := 0
	scanLines(charIn, func(line string) {
		// U+4E2D: zhōng,zhòng  # 中
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "U+") {
			return
		}
		body, _, _ := strings.Cut(line, "#")
		cpStr, rest, _ := strings.Cut(strings.TrimSpace(body), ":")
		cp, err := strconv.ParseUint(strings.TrimPrefix(cpStr, "U+"), 16, 32)
		if err != nil {
			return
		}
		r := rune(cp)
		if !isHan(r) {
			return
		}
		readings, _, _ := strings.Cut(strings.TrimSpace(rest), "  ")
		primary := true
		for _, py := range strings.Split(readings, ",") {
			p := tone.Normalize(py)
			if p == "" {
				continue
			}
			// 首读音（kMandarin 主要读音）保留全频；其余读音降为 1%，避免
			// "是 ti"、"有 wei" 等冷僻读音污染高频候选
			f := freq[string(r)]
			if !primary {
				f = max(f/100, 2)
			}
			add(string(r), p, max(f, 2))
			nChar++
			primary = false
		}
	})

	// 2) 词组：拼音数须与汉字数一致，全部为汉字
	nPhrase := 0
	scanLines(phraseIn, func(line string) {
		// 爱国: ài guó
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			return
		}
		w, pys, _ := strings.Cut(line, ":")
		w = strings.TrimSpace(w)
		if w == "" {
			return
		}
		runes := []rune(w)
		if len(runes) < 2 || len(runes) > 8 {
			return
		}
		for _, r := range runes {
			if !isHan(r) {
				return
			}
		}
		syls := strings.Fields(pys)
		if len(syls) != len(runes) {
			return
		}
		for i, py := range syls {
			p := tone.Normalize(py)
			if p == "" {
				return
			}
			syls[i] = p
		}
		add(w, strings.Join(syls, " "), max(freq[w], 20))
		nPhrase++
	})

	// 排序输出（词频降序，保证高频词在前、去重稳定）
	sort.Slice(order, func(i, j int) bool {
		if seen[order[i]] != seen[order[j]] {
			return seen[order[i]] > seen[order[j]]
		}
		if len(order[i].w) != len(order[j].w) {
			return len(order[i].w) < len(order[j].w)
		}
		return order[i].w < order[j].w
	})

	f, err := os.Create(out)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	bw := bufio.NewWriterSize(f, 1<<20)
	for _, k := range order {
		fmt.Fprintf(bw, "%s\t%s\t%d\n", k.w, k.p, seen[k])
	}
	if err := bw.Flush(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("完成: 单字 %d 条，词组 %d 条，合计 %d 条 → %s\n",
		nChar, nPhrase, len(order), out)
}

func loadFreq(path string, m map[string]int) {
	scanLines(path, func(line string) {
		cols := strings.Fields(line)
		if len(cols) < 2 {
			return
		}
		if n, err := strconv.Atoi(cols[1]); err == nil {
			m[cols[0]] = n
		}
	})
}

func scanLines(path string, fn func(string)) {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("打开 %s: %v", path, err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		fn(sc.Text())
	}
}
