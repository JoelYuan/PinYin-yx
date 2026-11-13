#!/usr/bin/env bash
# PinYin-yx 拼音查词 —— 双击或命令行启动（自动打开浏览器界面）
cd "$(dirname "$0")" || exit 1
exec ./bin/pinyin-web "$@"
