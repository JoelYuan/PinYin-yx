// pinyin-web 是 PinYin-yx 的桌面查词应用：本地 HTTP 服务 + 浏览器界面。
//
// 定位：一个应用界面内部调用离线词库的简单工具，不取代系统输入法。
// 敲拼音 → 实时候选 → 数字/点击选词 → 组句 → 一键复制到剪贴板。
//
// 纯 Go 标准库实现（无任何第三方依赖），双击脚本即可启动，
// 启动后自动打开浏览器窗口。
package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"pinyinyx/internal/dict"
	"pinyinyx/internal/engine"
)

//go:embed web/index.html
var indexHTML []byte

type server struct {
	eng *engine.Engine
}

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(indexHTML)
}

func (s *server) handleQuery(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Input string `json:"input"`
		N     int    `json:"n"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.N <= 0 || req.N > 9 {
		req.N = 9
	}
	cands := s.eng.Candidates(req.Input, req.N)
	type cand struct {
		Word string `json:"word"`
		Freq int    `json:"freq"`
	}
	out := make([]cand, 0, len(cands))
	for _, c := range cands {
		out = append(out, cand{Word: c.Word, Freq: c.Freq})
	}
	writeJSON(w, out)
}

func (s *server) handleLearn(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Word string `json:"word"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Word == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	s.eng.Store().Learn(req.Word)
	if err := s.eng.Store().SaveUser(); err != nil {
		log.Printf("保存用户词频失败: %v", err)
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	}
	if cmd != nil {
		_ = cmd.Start()
	}
}

func main() {
	addr := flag.String("addr", "127.0.0.1:8613", "监听地址")
	dictPath := flag.String("dict", "data/dict.txt", "主词典路径")
	userPath := flag.String("user", "data/user.txt", "用户词频路径")
	noOpen := flag.Bool("no-open", false, "不自动打开浏览器")
	flag.Parse()

	store, err := dict.Load(*dictPath)
	if err != nil {
		log.Fatalf("加载词典失败: %v", err)
	}
	store.LoadUser(*userPath)
	s := &server{eng: engine.New(store)}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/query", s.handleQuery)
	mux.HandleFunc("/api/learn", s.handleLearn)

	url := fmt.Sprintf("http://%s", *addr)
	log.Printf("PinYin-yx 拼音查词已启动: %s", url)
	if !*noOpen {
		go func() {
			time.Sleep(300 * time.Millisecond)
			openBrowser(url)
		}()
	}
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatal(err)
	}
}
