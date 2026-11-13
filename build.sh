#!/usr/bin/env bash
# 一键构建：本地调试版 + 发布版（linux-amd64，dist/ 下）
# 用法：./build.sh
set -e
cd "$(dirname "$0")"

# 本机 /tmp 过小时需要（普通机器可去掉这两行）
export GOTMPDIR="${GOTMPDIR:-$HOME/.cache/go-tmp}"
mkdir -p "$GOTMPDIR"

# 优先用本地安装的 Go（避免踩到损坏的 /snap/bin/go）
GO=${GO:-$(command -v "$HOME/sdk/go/bin/go" || command -v go)}

echo ">> 编译本地调试版 bin/"
mkdir -p bin
"$GO" build -o bin/pinyin-web  ./cmd/pinyin-web
"$GO" build -o bin/query       ./cmd/query
"$GO" build -o bin/builddict   ./cmd/builddict
"$GO" build -o bin/pinyin-demo ./cmd/pinyin-demo

echo ">> 交叉编译发布版 dist/（linux-amd64，静态无依赖）"
mkdir -p dist
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 "$GO" build -trimpath -ldflags "-s -w" \
    -o dist/pinyin-web-linux-amd64 ./cmd/pinyin-web

echo ">> 完成：bin/（本机调试）、dist/pinyin-web-linux-amd64（可上传 GitHub Releases）"
