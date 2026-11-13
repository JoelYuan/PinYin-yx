# PinYin-yx —— 离线拼音查词应用（Go · Ubuntu）

完全离线、可自主开发维护的拼音查词组句工具。核心引擎为纯 Go 库，
提供一个**独立窗口应用**（本地 HTTP 服务 + 浏览器界面，纯标准库零依赖）：
敲拼音 → 实时候选 → 数字/点击选词 → 组句 → 一键复制。
不取代系统输入法，只是应用界面内部调用词库的简单功能。

## 快速开始

```bash
./run.sh            # 双击亦可：启动服务并自动打开浏览器 http://127.0.0.1:8613
```

界面按键：`a-z`/`'` 输入拼音 → 数字 `1-9` 或点击候选 → 组句；
`空格` 选首选，`Enter` 将字母原样并入，`Esc` 清空，`Backspace` 删除。
选词记录写入 `data/user.txt`，越用越顺手。`复制组句` 按钮将结果放入剪贴板。

## 目录结构

```
PinYin-yx/
├── run.sh                 # ★ 双击启动脚本
├── cmd/
│   ├── pinyin-web/        # 桌面应用（HTTP 服务 + 内嵌网页，纯标准库）
│   ├── builddict/         # 词典构建器：合并开源数据源 → data/dict.txt
│   ├── pinyin-demo/       # 终端输入法演示（备选交互形态）
│   └── query/             # 候选查询工具（交叉核对用）
├── internal/
│   ├── tone/              # 声调归一化（zhōng→zhong, ü→v）
│   ├── dict/              # 词典加载/查询/用户词频学习
│   └── engine/            # 拼音切分（DP）、候选检索与排序
├── data/
│   ├── pinyin_data.txt    # 单字拼音（mozillazg/pinyin-data, MIT）
│   ├── phrase_pinyin.txt  # 词组拼音（mozillazg/phrase-pinyin-data, MIT）
│   ├── jieba_dict.txt     # 简体词频（fxsjy/jieba, MIT）
│   ├── test_dict.txt      # 单元测试夹具
│   └── dict.txt           # ★ 构建产物：44 万条主词典（词/拼音/词频）
├── build.sh               # 一键构建：bin/ 调试版 + dist/ 发布版
└── dist/                  # 发布二进制（静态链接，免依赖）
```

## 从 GitHub 获取（其他 Ubuntu 机器）

**方式 A —— 下载发布二进制（无需安装 Go）：**

到 [Releases](../../releases) 下载 `pinyin-web-linux-amd64`，然后：

```bash
chmod +x pinyin-web-linux-amd64
./pinyin-web-linux-amd64        # 静态链接，任何 glibc Linux 免依赖运行
```

**方式 B —— 源码构建：**

```bash
git clone <本仓库地址> && cd PinYin-yx
./build.sh          # 需要 Go 1.21+（构建脚本自动优先用 ~/sdk/go/bin/go）
./run.sh
```

主词典 `data/dict.txt` 已随仓库提交，克隆后无需重建即可直接使用；
仅当你更换了数据源时才需要重跑 `./bin/builddict`。
（`GOTMPDIR` 设置仅本机 `/tmp` 过小时需要，普通机器不必理会。）

## 核心算法

1. **声调归一化**：带声调拼音 → 键盘编码（`zhōng`→`zhong`，`ü`→`v`）。
2. **拼音切分**：对输入串做动态规划，枚举全部合法音节切分（每段限 16 种），
   `'` 强制音节边界；末尾未敲完的音节按前缀匹配补全（带 0.4 惩罚系数）。
3. **候选排序**：`score = ln(词频) + 用户词频×30`，完整匹配优先于前缀补全。
4. **多音字处理**：pinyin-data 首读音（普通话主要读音）保留全频，
   其余读音降为 1%，避免「是 ti」「有 wei」类冷僻读音污染候选。
5. **用户学习**：每次选词计数持久化，常用词自动升为首选。

## 系统级集成路线图

- **Phase 1（当前）**：核心引擎 + 终端演示 ✔
- **Phase 2**：逐字组词/整句生成（词库无整词时按字频组合，如 `nihao` 无整词时给出 你+好）
- **Phase 3**：IBus 引擎。IBus 协议基于 D-Bus，Go 用 `godbus` 可完整实现
  `org.freedesktop.IBus.Engine` 接口（ProcessKeyEvent → 转发到 engine 包，
  CommitText/UpdatePreedit 回写），无需 C 绑定。
- **Phase 4（可选）**：Fcitx5 对接。Fcitx5 官方插件仅支持 C++/Lua，Go 无法直接
  写插件；可行方案是 Fcitx5 的 DBus/IBus 前端模式加载本引擎。
- **Phase 5**：双拼/模糊音/整句智能纠错（参考 libpinyin 的 n-gram 模型）。

## 参考成功项目

| 项目 | 借鉴点 |
|------|--------|
| [Rime 中州韵](https://github.com/rime) | 词典格式、码表组织、开源数据源 |
| [libpinyin](https://github.com/libpinyin/libpinyin) | 切分与整句算法（后续 n-gram 参考） |
| [pinyin-data / phrase-pinyin-data](https://github.com/mozillazg) | 字词拼音数据（MIT） |
| [jieba](https://github.com/fxsjy/jieba) | 简体词频数据（MIT） |

## 数据更新

切换或更新数据源后重跑 `./bin/builddict` 即可；三个原始数据文件与脚本同库归档，
便于追溯。
