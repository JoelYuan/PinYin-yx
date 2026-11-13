package engine

import (
	"testing"

	"pinyinyx/internal/dict"
)

func TestCandidates(t *testing.T) {
	// 使用真实小词典
	s, err := dict.Load("../../data/test_dict.txt")
	if err != nil {
		t.Skipf("跳过：无测试词典 (%v)", err)
	}
	e := New(s)
	got := e.Candidates("nihao", 5)
	if len(got) == 0 {
		t.Fatal("nihao 应有候选")
	}
	if got[0].Word != "你好" {
		t.Fatalf("nihao 首选应为 你好，得到 %s", got[0].Word)
	}
	// 前缀匹配
	got = e.Candidates("ni'h", 5)
	found := false
	for _, g := range got {
		if g.Word == "拟好" {
			found = true
		}
	}
	if !found {
		t.Fatalf("ni'h 应包含 拟好，得到 %v", got)
	}
}
