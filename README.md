# Go-BBDown

🚀 **高性能哔哩哔哩视频下载器 - Go语言实现**

基于原C#版本BBDown的完整Go语言重写，提供更快的速度、更小的内存占用和更好的跨平台体验。

![Go Version](https://img.shields.io/badge/Go-1.21+-blue)
![License](https://img.shields.io/badge/License-MIT-green)
![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey)

> 本项目仅供个人学习、研究和非商业性用途。用户在使用本工具时，需自行确保遵守相关法律法规，特别是与版权相关的法律条款。

## ✨ 核心特性

### 🔐 完整的认证系统
- **WBI签名算法**: 完整实现哔哩哔哩最新WBI认证机制
- **多API支持**: Web端、TV端、APP端、国际版API
- **智能切换**: 自动选择最佳API接口获取高质量资源
- **Cookie管理**: 支持网页端和TV端Cookie认证

### ⚡ 高性能下载引擎
- **多线程并发**: Goroutine并发下载，最大化带宽利用
- **智能分块**: 自动计算最优分块大小
- **断点续传**: 支持下载中断后继续
- **进度监控**: 实时显示下载进度和速度

### 🎵 智能音视频处理
- **格式支持**: M4A、MP3、AAC、Opus、FLAC等音频格式
- **自动检测**: 智能识别音频编码格式
- **FFmpeg集成**: 无缝音视频混流处理
- **单独保存**: 音频文件自动单独保存

### 📁 高级文件管理
- **智能命名**: part为空时使用标题，否则使用part
- **多分P处理**: 自动添加_P1、_P2后缀避免重复
- **跨平台**: 统一的文件名处理和路径管理
- **模板支持**: 自定义文件名模板

### 🌐 全平台兼容
- **单文件部署**: 编译后单个可执行文件
- **跨平台**: Windows、macOS、Linux全支持
- **低资源**: 极低的内存占用和CPU使用
- **容器化**: 提供Docker部署方案

## 🏗️ 技术架构

### 核心模块设计

```
go-bbdown/
├── main.go                 # 程序入口点
├── cmd/                    # 命令行接口
│   └── root.go            # Cobra命令行框架
├── core/                  # 核心业务逻辑
│   ├── config.go         # 配置管理
│   ├── download.go       # 下载引擎
│   ├── entities.go       # 数据结构定义
│   ├── http.go          # HTTP客户端
│   └── parser.go        # 数据解析器
└── util/                 # 工具库
    ├── file.go          # 文件操作工具
    └── string.go        # 字符串处理工具
```

### 核心算法实现

#### WBI签名算法
```go
// 动态获取WBI密钥并生成签名
func GenerateWBISign(params map[string]string) string {
    mixKey := getMixKey()
    // 按哔哩哔哩规则生成签名
    return wbiSignature
}
```

#### BV转换算法
```go
// 高效的BV号与AV号双向转换
func BvToAv(bvid string) (int64, error) {
    // XOR变换和表映射算法
    return aid, nil
}
```

#### 多线程下载
```go
// Goroutine并发下载实现
func multiThreadDownload(url, filename string, threads int) error {
    // 智能分块和并发管理
    return downloadWithProgress()
}
```

## 🚀 快速开始

### 安装方式

#### 从源码编译
```bash
git clone https://github.com/tekintian/go-bbdown.git
cd go-bbdown
go mod tidy
go build -o bbdown main.go
```

#### 预编译下载（推荐）
```bash
# 下载对应平台的可执行文件
wget https://github.com/tekintian/go-bbdown/releases/latest/download/bbdown-linux-amd64
chmod +x bbdown-linux-amd64
```

### 基本使用

```bash
# 下载普通视频
./bbdown https://www.bilibili.com/video/BV1xxxxxx

# 使用TV接口（高质量无水印）
./bbdown -tv https://www.bilibili.com/video/BV1xxxxxx

# 多线程下载
./bbdown -mt https://www.bilibili.com/video/BV1xxxxxx

# 交互式选择清晰度
./bbdown -ia https://www.bilibili.com/video/BV1xxxxxx
```

## 📋 命令行参数详解

### API模式选择
- `-tv, --use-tv-api` - 使用TV端API（高质量无水印）
- `-app, --use-app-api` - 使用APP端API
- `-intl, --use-intl-api` - 使用国际版API（东南亚视频）

### 下载控制
- `-mt, --multi-thread` - 启用多线程下载（默认开启）
- `-ia, --interactive` - 交互式选择清晰度
- `--video-only` - 仅下载视频流
- `--audio-only` - 仅下载音频流
- `--skip-mux` - 跳过音视频混流

### 质量选择
- `-e, --encoding-priority` - 视频编码优先级："hevc,av1,avc"
- `-q, --dfn-priority` - 画质优先级："8K 超高清, 4K 超清, 1080P 高码率"

### 文件管理
- `-F, --file-pattern` - 单文件命名模板
- `-M, --multi-file-pattern` - 多文件命名模板
- `-p, --select-page` - 选择指定分P："1,3-5" 或 "ALL"

### 网络设置
- `-c, --cookie` - 网页端Cookie
- `-token, --access-token` - TV/APP端访问令牌
- `--ffmpeg-path` - FFmpeg可执行文件路径
- `--work-dir` - 设置工作目录

## 🎯 高级使用场景

### 智能文件命名
```bash
# 自定义单文件命名
./bbdown -F "<videoTitle>_<dfn>" https://www.bilibili.com/video/BV1xxxxxx

# 多分P文件组织
./bbdown -M "<videoTitle>/P<pageNumberWithZero>_<pageTitle>" https://www.bilibili.com/video/BV1xxxxxx
```

### 认证配置
```bash
# 使用网页Cookie下载会员内容
./bbdown -c "SESSDATA=xxx; bili_jct=xxx" https://www.bilibili.com/video/BV1xxxxxx

# 使用TV端Token
./bbdown -tv -token "access_token=xxx" https://www.bilibili.com/video/BV1xxxxxx

# 使用APP端Token
./bbdown -app -token "access_token=xxx" https://www.bilibili.com/video/BV1xxxxxx
```

### 分P下载控制
```bash
# 下载指定分P
./bbdown -p 1,3,5 https://www.bilibili.com/video/BV1xxxxxx

# 下载范围分P
./bbdown -p 1-10 https://www.bilibili.com/video/BV1xxxxxx

# 下载所有分P
./bbdown -p ALL https://www.bilibili.com/video/BV1xxxxxx
```

### 仅音频下载
```bash
# 下载高质量音频
./bbdown --audio-only -q "320kbps" https://www.bilibili.com/video/BV1xxxxxx

# 自动保存音频文件（格式检测）
./bbdown https://www.bilibili.com/video/BV1xxxxxx
# 会同时保存：video.mp4 和 audio.m4a
```

## 📊 性能对比

| 特性 | C#版本 | Go版本 |
|------|--------|--------|
| 启动速度 | 慢（.NET启动） | 快 |
| 内存占用 | 高（~100MB） | 低（~20MB） |
| CPU使用 | 中等 | 低 |
| 并发性能 | 中等 | 优秀 |
| 文件大小 | 大（~50MB） | 小（~15MB） |
| 部署方式 | 依赖.NET运行时 | 单文件部署 |
| 跨平台 | 良好 | 优秀 |

## 🔧 外部依赖

### 必需依赖
- **FFmpeg**: 音视频混流（推荐5.0+）
  - Windows: 下载到 `C:\ffmpeg\bin\ffmpeg.exe`
  - macOS: `brew install ffmpeg`
  - Linux: `sudo apt install ffmpeg`

### 可选依赖
- **aria2c**: 多线程下载加速器
  - Windows: 下载并添加到PATH
  - macOS: `brew install aria2`
  - Linux: `sudo apt install aria2c`

### 依赖检测
程序启动时会自动检测依赖：
```bash
./bbdown
[INFO] FFmpeg found: /usr/local/bin/ffmpeg
[INFO] aria2c found: /usr/local/bin/aria2c
[INFO] All dependencies ready
```

## 📝 配置文件

支持YAML配置文件，位置：`~/.bbdown.yaml`

```yaml
# API设置
api:
  use-tv: false
  use-app: false
  use-intl: false

# 下载设置
download:
  multi-thread: true
  encoding-priority: ["hevc", "av1", "avc"]
  dfn-priority: ["8K 超高清", "4K 超清", "1080P 高码率"]

# 文件设置
files:
  file-pattern: "<videoTitle>"
  multi-file-pattern: "<videoTitle>/[P<pageNumberWithZero>]<pageTitle>"
  work-dir: "./downloads"

# 外部工具
tools:
  ffmpeg-path: "/usr/local/bin/ffmpeg"
  aria2c-path: "/usr/local/bin/aria2c"

# 网络设置
network:
  user-agent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
  cookie: ""
  access-token: ""
```

## 🐳 Docker部署

```dockerfile
FROM alpine:latest
RUN apk add --no-cache ffmpeg aria2
COPY bbdown /usr/local/bin/
ENTRYPOINT ["bbdown"]
```

使用方法：
```bash
docker build -t bbdown .
docker run -v $(pwd)/downloads:/downloads bbdown https://www.bilibili.com/video/BV1xxxxxx
```

## 🎮 实际应用示例

### 1. 批量下载系列视频
```bash
# 创建脚本
for url in $(cat video_list.txt); do
    ./bbdown -tv -mt "$url"
done
```

### 2. 高质量音频提取
```bash
# 提取无损音频
./bbdown --audio-only -q "无损" https://www.bilibili.com/video/BV1xxxxxx
```

### 3. 自动化工作流
```bash
#!/bin/bash
# 下载后自动上传到云存储
URL="https://www.bilibili.com/video/BV1xxxxxx"
./bbdown -tv "$URL"
find . -name "*.mp4" -exec rclone copy {} cloud:videos/ \;
```

## 🐛 故障排除

### 常见问题

**Q: 提示"FFmpeg not found"**
```bash
# 解决方案1：指定路径
./bbdown --ffmpeg-path /path/to/ffmpeg https://...

# 解决方案2：添加到PATH
export PATH=$PATH:/path/to/ffmpeg/dir
```

**Q: 下载速度慢**
```bash
# 使用TV端API（通常更快）
./bbdown -tv -mt https://...

# 或使用aria2c加速
./bbdown --use-aria2c https://...
```

**Q: 无法下载会员内容**
```bash
# 检查Cookie是否有效
./bbdown -c "SESSDATA=xxx" --info https://...
# 确认能看到会员清晰度后再下载
```

### 调试模式
```bash
# 启用详细日志
./bbdown --debug https://www.bilibili.com/video/BV1xxxxxx

# 仅查看信息，不下载
./bbdown --info https://www.bilibili.com/video/BV1xxxxxx
```

## 🔄 版本更新

```bash
# 检查当前版本
./bbdown --version

# 更新到最新版本
go install github.com/tekintian/go-bbdown/go_bbdown@latest
```

## 🤝 贡献指南

欢迎贡献代码！请遵循以下步骤：

1. Fork本项目
2. 创建功能分支：`git checkout -b feature/amazing-feature`
3. 提交更改：`git commit -m 'Add amazing feature'`
4. 推送分支：`git push origin feature/amazing-feature`
5. 提交Pull Request

## 📄 许可证

本项目基于MIT许可证开源。详见[LICENSE](../LICENSE)文件。

## 🙏 致谢

- 原C#版本BBDown：[nilaoda/BBDown](https://github.com/nilaoda/BBDown)
- 哔哩哔哩API收集：[SocialSisterYi/bilibili-API-collect](https://github.com/SocialSisterYi/bilibili-API-collect)
- Cobra框架：[spf13/cobra](https://github.com/spf13/cobra)

---

**Go-BBDown**: 为追求极致性能而生的哔哩哔哩下载器 🚀