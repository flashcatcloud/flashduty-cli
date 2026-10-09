# Flashduty CLI

[English](README.md) | 中文

[![License](https://img.shields.io/github/license/flashcatcloud/flashduty-cli?style=flat-square&color=24bfa5&label=License)](LICENSE)
[![Release](https://img.shields.io/github/v/release/flashcatcloud/flashduty-cli?style=flat-square&color=24bfa5)](https://github.com/flashcatcloud/flashduty-cli/releases)
[![CI](https://img.shields.io/github/actions/workflow/status/flashcatcloud/flashduty-cli/ci.yml?style=flat-square&branch=main&label=CI)](https://github.com/flashcatcloud/flashduty-cli/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/flashcatcloud/flashduty-cli?style=flat-square)](https://goreportcard.com/report/github.com/flashcatcloud/flashduty-cli)

**Flashduty CLI**（`flashduty`）是 [Flashduty](https://www.flashduty.com) 故障管理与值班平台的官方开源命令行工具。在终端、Shell 脚本或 AI 编程助手里，都可以用它处理故障和告警、查询值班、发布状态页更新、管理监控和 RUM，以及调用 AI SRE。

[官网](https://www.flashduty.com) · [CLI 文档](https://docs.flashduty.com/zh/developer/cli) · [API 参考](https://docs.flashduty.com/zh/openapi/introduction) · [控制台](https://console.flashcat.cloud) · [博客：写给人和 Agent 的 CLI](https://www.flashduty.com/zh/now/blog/flashduty-cli) · [版本发布](https://github.com/flashcatcloud/flashduty-cli/releases)

## 特点

- **覆盖全部公开 API。** 每个公开的 Flashduty API 都有对应命令，由 OpenAPI 规范经 [go-flashduty](https://github.com/flashcatcloud/go-flashduty) SDK 生成。故障、告警、值班、状态页等常用流程另有手写命令，flag 更短，表格更易读。
- **命令名可预测。** API 路径直接对应命令：`POST /incident/merge` 是 `flashduty incident merge`，`POST /status-page/change/create` 是 `flashduty status-page change-create`。
- **适合脚本和 Agent。** 输出支持 `table`、`json`、`toon`（紧凑格式，更省 token）。列表分页有大小上限，被裁剪时会明确标出。`--fields` 只返回需要的字段。
- **单个二进制。** 支持 macOS、Linux、Windows 的 amd64 和 arm64。`flashduty update` 原地升级。

## 安装

### macOS / Linux

```bash
curl -sSL https://static.flashcat.cloud/flashduty-cli/install.sh | sh
```

### Windows (PowerShell)

```powershell
irm https://static.flashcat.cloud/flashduty-cli/install.ps1 | iex
```

### 手动下载

从 [GitHub Releases](https://github.com/flashcatcloud/flashduty-cli/releases) 下载对应平台的压缩包，解压后把二进制放到 `PATH` 中。

### 安装选项

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `FLASHDUTY_VERSION` | 安装指定版本（如 `v1.5.12`） | 最新版 |
| `FLASHDUTY_INSTALL_DIR` | 安装目录 | `/usr/local/bin`（shell），`~\.flashduty\bin`（PowerShell） |
| `MIRROR_URL` | 下载镜像地址（必须是 `https://`） | `https://static.flashcat.cloud/flashduty-cli` |

## 快速上手

```bash
# 1. 用 APP Key 登录（控制台：我的 → APP Key）
flashduty login
flashduty whoami

# 2. 处理故障
flashduty incident list --since 24h --severity Critical
flashduty incident info <incident_id>
flashduty incident ack <incident_id>
flashduty incident merge <target_id> --source <id1>,<id2>   # 源故障默认关闭并保留
flashduty incident close <incident_id>

# 3. 谁在值班，最近有什么变更
flashduty oncall who
flashduty change list --since 2h

# 4. 浏览任意模块
flashduty status-page --help
```

APP Key 的创建方法见 [API 参考](https://docs.flashduty.com/zh/openapi/introduction)。

## 命令分组

`flashduty <分组> --help` 列出分组内的命令，`flashduty <分组> <命令> --help` 查看 flag 和示例。常用流程见 [CLI 文档](https://docs.flashduty.com/zh/developer/cli)。

| 模块 | 分组 |
|------|------|
| On-call | `incident`、`alert`、`alert-event`、`change`、`channel`、`route`、`oncall`、`schedule`、`calendar`、`integration`、`webhook`、`enrichment`、`field`、`template`、`insight`、`status-page` |
| 监控 | `monit`、`monit-query`、`datasource` |
| RUM | `rum`、`sourcemap` |
| AI SRE | `safari`、`session`、`automation` |
| 平台 | `account`、`member`、`person`、`team`、`role`、`audit` |
| CLI 自身 | `login`、`whoami`、`config`、`update`、`version`、`completion` |

### 请求体

生成的命令把请求的每个顶层字段做成带类型的 flag，完整 JSON 请求体通过 `--data` 传入（`--data -` 从 stdin 读取）。位置参数和 flag 会覆盖 `--data` 中的同名字段，所以嵌套对象和数组放进 `--data`，标量字段用 flag：

```bash
flashduty status-page change-create <page_id> --type incident \
  --title "API latency elevated" --status investigating \
  --data '{"updates":[{"status":"investigating","description":"Investigating.","component_changes":[{"component_id":"<component_id>","status":"degraded"}]}]}'
```

## 认证与配置

凭据按以下顺序读取：

1. `--app-key` flag（隐藏，供脚本使用）
2. `FLASHDUTY_APP_KEY` 环境变量
3. `~/.flashduty/config.yaml`，由 `flashduty login` 写入，权限 `0600`

```yaml
app_key: your_app_key
base_url: https://api.flashcat.cloud
```

```bash
flashduty config show              # 打印当前配置（key 已脱敏）
flashduty config set app_key KEY   # 设置 APP Key
flashduty config set base_url URL  # 修改 API 地址
```

| 环境变量 | 用途 |
|----------|------|
| `FLASHDUTY_APP_KEY` | APP Key |
| `FLASHDUTY_BASE_URL` | API 地址（默认 `https://api.flashcat.cloud`） |
| `FLASHDUTY_NO_UPDATE_CHECK=1` | 关闭每天一次的后台更新检查 |
| `FLASHDUTY_UPDATE_BASE_URL` | `flashduty update` 和更新检查使用的镜像地址 |

## 全局 flag

| Flag | 说明 |
|------|------|
| `--output-format` | `table`（默认）、`json` 或 `toon` |
| `--json` | 等同 `--output-format json` |
| `--no-trunc` | 表格输出不截断长字段 |
| `--fields` | `json`/`toon` 输出只保留这些顶层字段（逗号分隔）：列表按行保留，单条记录保留对应 key |
| `--base-url` | 覆盖 API 地址 |
| `--version` | 打印版本（同 `flashduty version`） |

## 输出格式

- **Table**（默认）：对齐的列，给人看；长字段会截断，加 `--no-trunc` 不截断。
- **JSON**（`--json`）：给 `jq` 和脚本用，例如 `flashduty incident list --json | jq '.[].title'`。
- **TOON**（`--output-format toon`）：[Token-Oriented Object Notation](https://github.com/toon-format/toon-go)。JSON 每行都重复字段名，TOON 不重复，列表输出的 token 少得多。LLM 或 Agent 读取输出时用它。

每一页结构化列表最大 16 KiB。被裁剪的页会在 stderr 上提示，列表的返回体里也会带 `"truncated": true`。同时带 `"emitted_rows": N` 表示只返回了前 N 行：用更小的 `--limit` 重新请求，直到拿到的行数达到 `total`。不带 `emitted_rows` 表示行都在，但长字段被截断：用 `--fields` 缩小字段范围。

按时间范围查询的命令会在 stderr 上打印实际发出的时间窗口（本地时区），例如 `note: window 2026-10-08T14:00:00+08:00..2026-10-09T14:00:00+08:00 (1d, ended now)`；stdout 仍是纯 `json`/`toon`。`incident list --progress Triggered,Processing` 和 `alert list --active` 在没给 `--since` 时默认查最近 30 天，因为未关闭的记录可能早于一天。

未知 flag 或命令的报错会列出最接近的有效名称，以及该层级所有可用的选项。

## 升级

```bash
flashduty update           # 原地安装最新版
flashduty update --check   # 只检查是否有新版本
```

## 开发

需要 Go 1.26+（见 `go.mod`）。golangci-lint 由 Makefile 自动安装。

```bash
make build        # 构建 bin/flashduty
make test         # 运行测试（带 race 检测）
make check        # fmt、lint、test、build
make gen-cards    # 重新生成 skills/flashduty/reference 中的命令片段
make check-cards  # 用真实命令树校验这些片段
make help         # 全部目标
```

生成的命令在 `internal/cli/zz_generated_*.go`，由 `go run ./internal/cmd/cligen` 根据 go-flashduty 自带的 OpenAPI 规范生成；手写命令放在同一目录。手写命令占用了生成命令的名字时，`TestCuratedCommandsCoverRequestFields` 要求它仍然能设置该 API 的每个请求字段。

| 依赖 | 用途 |
|------|------|
| [go-flashduty](https://github.com/flashcatcloud/go-flashduty) | Flashduty API 客户端，由 OpenAPI 规范生成 |
| [cobra](https://github.com/spf13/cobra) | 命令框架 |
| [toon-go](https://github.com/toon-format/toon-go) | TOON 输出 |
| [yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3) | 配置文件 |
| [x/term](https://pkg.go.dev/golang.org/x/term) | APP Key 隐藏输入 |

## 相关项目

- [go-flashduty](https://github.com/flashcatcloud/go-flashduty)：Flashduty API 的 Go SDK
- [flashduty-mcp-server](https://github.com/flashcatcloud/flashduty-mcp-server)：Flashduty 的 MCP Server
- [terraform-provider-flashduty](https://github.com/flashcatcloud/terraform-provider-flashduty)：管理 Flashduty 资源的 Terraform Provider

## 参与贡献

欢迎贡献。提交 PR 前请阅读 [CONTRIBUTING.md](CONTRIBUTING.md)，并遵守[行为准则](CODE_OF_CONDUCT.md)。

- [报告问题或提需求](https://github.com/flashcatcloud/flashduty-cli/issues/new/choose)
- [获取帮助](SUPPORT.md)
- [报告安全漏洞](SECURITY.md)

## 许可证

MIT，详见 [LICENSE](LICENSE)。
