// Package dict 提供拼音词典的加载、查询与用户词频学习。
//
// 词典文件格式（TSV，UTF-8，无 BOM）:
//
	// 词<TAB>拼音1 拼音2 ...（无声调，ü 记作 v）<TAB>词频
package dict

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// Entry 是一条词典候选。
type Entry struct {
	Word      string   // 汉字词
	Syllables []string // 无声调拼音串
	Freq      int      // 基础词频
}

// Store 持有全部词条及索引。
type Store struct {
	byKey    map[string][]*Entry // "ni hao" -> entries
	syllSet  map[string]struct{} // 词典中出现的全部音节
	sylList  []string            // 排序后的音节列表（供前缀匹配）
	userFreq map[string]int      // 用户自学习词频
	userPath string
}

// Load 从 TSV 文件加载词典。
func Load(path string) (*Store, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	s := &Store{
		byKey:    make(map[string][]*Entry, 1<<20),
		syllSet:  make(map[string]struct{}, 512),
		userFreq: make(map[string]int),
	}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		cols := strings.Split(line, "\t")
		if len(cols) < 2 {
			continue
		}
		syl := strings.Fields(cols[1])
		if len(syl) == 0 {
			continue
		}
		freq := 0
		if len(cols) >= 3 {
			freq, _ = strconv.Atoi(cols[2])
		}
		ent := &Entry{Word: cols[0], Syllables: syl, Freq: freq}
		key := strings.Join(syl, " ")
		s.byKey[key] = append(s.byKey[key], ent)
		for _, p := range syl {
			s.syllSet[p] = struct{}{}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	for p := range s.syllSet {
		s.sylList = append(s.sylList, p)
	}
	// 按长度降序、字典序排序，便于前缀扫描
	for i := 0; i < len(s.sylList); i++ {
		for j := i + 1; j < len(s.sylList); j++ {
			a, b := s.sylList[i], s.sylList[j]
			if len(a) < len(b) || (len(a) == len(b) && a > b) {
				s.sylList[i], s.sylList[j] = b, a
			}
		}
	}
	return s, nil
}

// Lookup 返回与音节序列完全匹配的词条。
func (s *Store) Lookup(syllables []string) []*Entry {
	return s.byKey[strings.Join(syllables, " ")]
}

// Completions 返回以 prefix 开头的全部音节。
func (s *Store) Completions(prefix string) []string {
	var out []string
	for _, p := range s.sylList {
		if strings.HasPrefix(p, prefix) {
			out = append(out, p)
		}
	}
	return out
}

// Syllables 返回词典音节集合。
func (s *Store) Syllables() map[string]struct{} { return s.syllSet }

// Learn 记录一次用户选词，提升该词后续排序。
func (s *Store) Learn(word string) {
	s.userFreq[word]++
}

// UserFreq 返回用户词频。
func (s *Store) UserFreq(word string) int { return s.userFreq[word] }

// LoadUser 加载用户词频文件（词<TAB>次数）。文件不存在时仅记录路径，
// 供后续 SaveUser 首次写出。
func (s *Store) LoadUser(path string) {
	s.userPath = path
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		cols := strings.Split(strings.TrimSpace(sc.Text()), "\t")
		if len(cols) != 2 {
			continue
		}
		if n, err := strconv.Atoi(cols[1]); err == nil {
			s.userFreq[cols[0]] += n
		}
	}
}

// SaveUser 持久化用户词频。
func (s *Store) SaveUser() error {
	if s.userPath == "" {
		return nil
	}
	f, err := os.Create(s.userPath)
	if err != nil {
		return err
	}
	defer f.Close()
	for w, n := range s.userFreq {
		if _, err := f.WriteString(w + "\t" + strconv.Itoa(n) + "\n"); err != nil {
			return err
		}
	}
	return nil
}
