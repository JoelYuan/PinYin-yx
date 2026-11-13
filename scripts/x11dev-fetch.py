#!/usr/bin/env python3
"""x11dev-fetch.py — 免 root 获取 X11 GUI 开发包（供 Gio cgo 编译用）。

从 Ubuntu noble 镜像的 Packages 索引解析目标包及其递归依赖的 .deb 路径，
用 curl 下载后解压到用户目录（PREFIX），无需 sudo。

用法: python3 x11dev-fetch.py
输出: $HOME/.cache/x11root/ 下含 include/ 与 lib/（.pc 文件 prefix 已修正）
之后编译时设置: PKG_CONFIG_PATH=$HOME/.cache/x11root/lib/x86_64-linux-gnu/pkgconfig:$HOME/.cache/x11root/lib/pkgconfig
"""
import gzip
import os
import subprocess
import sys
import urllib.request

MIRROR = "http://mirrors.tuna.tsinghua.edu.cn/ubuntu"
INDEX = os.path.expanduser("~/.cache/x11dev/Packages.gz")
PREFIX = os.path.expanduser("~/.cache/x11root")

ROOTS = [
    "libx11-dev", "libx11-xcb-dev", "libxkbcommon-dev", "libxkbcommon-x11-dev",
    "libxcursor-dev", "libxfixes-dev",
]

def load_index():
    pkgs = {}
    cur = {}
    with gzip.open(INDEX, "rt", errors="replace") as f:
        for line in f:
            if line == "\n":
                name = cur.get("Package")
                if name and name not in pkgs:
                    pkgs[name] = cur
                cur = {}
            else:
                if ": " in line:
                    k, v = line.split(": ", 1)
                    cur[k] = v.strip()
    return pkgs

def deps_of(pkg, pkgs):
    out = []
    for field in ("Depends",):
        val = pkg.get(field, "")
        for part in val.split(","):
            first = part.split("|")[0].strip()
            name = first.split()[0] if first.split() else None
            if name:
                out.append(name)
    return out

def resolve(pkgs, roots):
    seen, queue, missing = set(), list(roots), []
    order = []
    while queue:
        name = queue.pop(0)
        if name in seen:
            continue
        seen.add(name)
        pkg = pkgs.get(name)
        if pkg is None:
            missing.append(name)
            continue
        order.append(pkg)
        queue.extend(deps_of(pkg, pkgs))
    return order, missing

def extract_deb(dest):
    """纯 Python 解包 .deb（ar + tar），不设置 mtime，避开沙盒 utime 限制。"""
    import io
    import tarfile
    with open(dest, "rb") as f:
        blob = f.read()
    # 解析 ar 归档，找 data.tar.*
    pos = 8  # 跳过 "!<arch>\n"
    data_member = None
    while pos < len(blob):
        hdr = blob[pos:pos + 60]
        if len(hdr) < 60:
            break
        name = hdr[0:16].decode().strip()
        size = int(hdr[48:58].decode().strip())
        body = blob[pos + 60:pos + 60 + size]
        pos += 60 + size
        if pos % 2:
            pos += 1
        if name.startswith("data.tar"):
            data_member = body
    if data_member is None:
        raise RuntimeError("deb 中未找到 data.tar")
    with tarfile.open(fileobj=io.BytesIO(data_member)) as tar:
        for m in tar.getmembers():
            target = os.path.join(PREFIX, m.name)
            if m.isdir() or m.name.endswith("/"):
                os.makedirs(target, exist_ok=True)
            elif m.issym():
                os.makedirs(os.path.dirname(target), exist_ok=True)
                try:
                    os.unlink(target)
                except FileNotFoundError:
                    pass
                os.symlink(m.linkname, target)
            elif m.isfile():
                os.makedirs(os.path.dirname(target), exist_ok=True)
                src = tar.extractfile(m)
                with open(target, "wb") as out:
                    out.write(src.read())

def download_and_extract(pkgs_list):
    os.makedirs(PREFIX, exist_ok=True)
    debs = os.path.expanduser("~/.cache/x11dev/debs")
    os.makedirs(debs, exist_ok=True)
    ok, fail = 0, []
    for pkg in pkgs_list:
        fn = os.path.basename(pkg["Filename"])
        dest = os.path.join(debs, fn)
        if not os.path.exists(dest):
            url = MIRROR + "/" + pkg["Filename"]
            try:
                urllib.request.urlretrieve(url, dest)
            except Exception as e:
                fail.append((pkg["Package"], str(e)))
                continue
        extract_deb(dest)
        ok += 1
    return ok, fail

def fix_pkgconfig():
    import glob
    pcs = glob.glob(PREFIX + "/usr/lib/*/pkgconfig/*.pc") + \
          glob.glob(PREFIX + "/usr/lib/pkgconfig/*.pc") + \
          glob.glob(PREFIX + "/usr/share/pkgconfig/*.pc")
    for pc in pcs:
        with open(pc, "r", errors="replace") as f:
            txt = f.read()
        new = txt.replace("prefix=/usr", "prefix=" + PREFIX + "/usr")
        if new != txt:
            with open(pc, "w") as f:
                f.write(new)
    return len(pcs)

def link_runtime_libs():
    """-dev 包里的 .so 符号链接指向的运行库不在 prefix 中，
    将系统同名运行库软链进来以便链接器找到。"""
    import glob
    import re
    syslib = "/usr/lib/x86_64-linux-gnu"
    runtime = os.listdir(syslib)
    n = 0
    for so in glob.glob(PREFIX + "/usr/lib/*/*.so"):
        if os.path.exists(os.path.realpath(so)):
            continue  # prefix 内已存在真实文件
        name = os.path.basename(so)  # 如 libXcursor.so
        for f in runtime:
            if re.match(re.escape(name) + r"\.so\.\d+(\.\d+)*$", f):
                os.unlink(so)
                os.symlink(os.path.join(syslib, f), so)
                n += 1
                break
    return n

def main():
    pkgs = load_index()
    print(f"索引包数: {len(pkgs)}")
    order, missing = resolve(pkgs, ROOTS)
    print(f"需处理包: {len(order)}，缺失: {missing}")
    # 只解压 -dev/-dev 相关与头文件包；运行库包也解压无害
    ok, fail = download_and_extract(order)
    print(f"下载解压成功 {ok} 个，失败 {fail}")
    n_pc = fix_pkgconfig()
    n_so = link_runtime_libs()
    print(f"修正 .pc 文件 {n_pc} 个，链接运行库 {n_so} 个")
    print("PKG_CONFIG_PATH 建议:")
    print(f"{PREFIX}/lib/x86_64-linux-gnu/pkgconfig:{PREFIX}/lib/pkgconfig")

if __name__ == "__main__":
    main()
