// Package engine 实现拼音切分、候选检索与排序。
package engine

import (
	"math"
	"sort"
	"strings"

	"pinyinyx/internal/dict"
)

const (
	maxSegmentations = 16  // 单段最大切分数
	partialPenalty   = 0.4 // 未敲完音节的候选惩罚
	userBoost        = 30  // 用户词频权重
)

// Engine 是拼音输入引擎。
type Engine struct {
	store *dict.Store
}

// New 创建引擎。
func New(store *dict.Store) *Engine {
	return &Engine{store: store}
}

// Store 暴露底层词典（用于学习持久化等）。
func (e *Engine) Store() *dict.Store { return e.store }

// segmentOne 对单段（无分隔符）输入做动态规划切分，返回全部合法切分。
func segmentOne(part string, set map[string]struct{}, max int) [][]string {
	n := len(part)
	segs := make([][][]string, n+1)
	segs[0] = [][]string{{}}
	for i := 1; i <= n; i++ {
		var cur [][]string
		for j := 0; j < i && len(cur) < max; j++ {
			if len(segs[j]) == 0 {
				continue
			}
			if _, ok := set[part[j:i]]; !ok {
				continue
			}
			for _, prev := range segs[j] {
				if len(cur) >= max {
					break
				}
				ns := make([]string, len(prev)+1)
				copy(ns, prev)
				ns[len(prev)] = part[j:i]
				cur = append(cur, ns)
			}
		}
		segs[i] = cur
	}
	return segs[n]
}

// segmentations 处理以 ' 分隔的完整输入，返回各段切分的笛卡尔积（限量）。
func (e *Engine) segmentations(input string) [][]string {
	parts := strings.Split(strings.ToLower(input), "'")
	cur := [][]string{{}}
	for _, part := range parts {
		if part == "" {
			continue
		}
		segs := segmentOne(part, e.store.Syllables(), maxSegmentations)
		if len(segs) == 0 {
			return nil
		}
		var next [][]string
		for _, a := range cur {
			for _, b := range segs {
				if len(next) >= maxSegmentations {
					break
				}
				m := make([]string, 0, len(a)+len(b))
				m = append(m, a...)
				m = append(m, b...)
				next = append(next, m)
			}
		}
		cur = next
	}
	return cur
}

type scored struct {
	ent *dict.Entry
	s   float64
}

// Candidates 返回按分数排序的前 n 个候选。
// 规则：完全匹配优先；最后一段允许是未敲完的音节前缀（带惩罚）；
// 分数 = log(词频) + 用户词频加成 × 惩罚系数。
func (e *Engine) Candidates(input string, n int) []*dict.Entry {
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return nil
	}
	seen := make(map[string]bool, n)
	var pool []scored

	add := func(syllables []string, penalty float64) {
		for _, ent := range e.store.Lookup(syllables) {
			if seen[ent.Word] {
				continue
			}
			seen[ent.Word] = true
			s := math.Log(float64(ent.Freq+1)) +
				float64(e.store.UserFreq(ent.Word))*userBoost
			pool = append(pool, scored{ent, s * penalty})
		}
	}

	// 1) 完整切分匹配
	for _, seg := range e.segmentations(input) {
		add(seg, 1.0)
	}

	// 2) 末尾未敲完的音节前缀匹配（最长优先，找到即止）
	parts := strings.Split(input, "'")
	last := parts[len(parts)-1]
	for l := 1; l <= len(last); l++ {
		prefix := last[len(last)-l:]
		head := input[:len(input)-l]
		headSegs := e.segmentations(head)
		if len(headSegs) == 0 {
			continue
		}
		coms := e.store.Completions(prefix)
		if len(coms) == 0 {
			continue
		}
		for _, seg := range headSegs {
			for _, c := range coms {
				add(append(append([]string{}, seg...), c), partialPenalty)
			}
		}
		break
	}

	sort.SliceStable(pool, func(i, j int) bool { return pool[i].s > pool[j].s })
	if len(pool) > n {
		pool = pool[:n]
	}
	out := make([]*dict.Entry, len(pool))
	for i, p := range pool {
		out[i] = p.ent
	}
	return out
}

// Choose 记录用户选词。
func (e *Engine) Choose(ent *dict.Entry) {
	e.store.Learn(ent.Word)
}
