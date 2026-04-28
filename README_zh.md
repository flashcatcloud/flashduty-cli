# Flashduty CLI

[English](README.md) | 中文

[![License](https://img.shields.io/github/license/flashcatcloud/flashduty-cli?style=flat-square&color=24bfa5&label=License)](LICENSE)

[Flashduty](https://flashcat.cloud) 平台的命令行工具。在终端中管理故障、值班、状态页等。

## 安装

### macOS / Linux

```bash
curl -sSL https://raw.githubusercontent.com/flashcatcloud/flashduty-cli/main/install.sh | sh
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/flashcatcloud/flashduty-cli/main/install.ps1 | iex
```

### Go Install

```bash
go install github.com/flashcatcloud/flashduty-cli/cmd/flashduty@latest
```

> 确保 `$(go env GOPATH)/bin` 在您的 `PATH` 中。如果安装后找不到 `flashduty`，请运行：
> ```bash
> export PATH="$(go env GOPATH)/bin:$PATH"
> ```

### 手动下载

从 [GitHub Releases](https://github.com/flashcatcloud/flashduty-cli/releases) 下载适合您平台的最新版本。

### 选项

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `FLASHDUTY_VERSION` | 安装指定版本（如 `v0.1.2`） | 最新版 |
| `FLASHDUTY_INSTALL_DIR` | 自定义安装目录 | `/usr/local/bin`（Shell）、`~\.flashduty\bin`（PowerShell） |

## Agent Skills（AI 代理技能）

Flashduty CLI 内置 10 个代理技能，让 AI 编程代理能够通过 CLI 操作 Flashduty 平台。兼容 41+ 编程代理，包括 Claude Code、Cursor、GitHub Copilot、Codex、Gemini CLI、Windsurf 等。

```bash
npx skills add flashcatcloud/flashduty-cli -y -g
```

安装器会自动检测已安装的代理并为其安装技能。

### 可用技能

| 技能 | 范围 |
|------|------|
| `flashduty-shared` | 基础：认证、三层降噪模型、全局参数、安全规则 |
| `flashduty-incident` | 故障生命周期：分诊、调查、解决、合并、暂停、转派 |
| `flashduty-alert` | 告警与告警事件调查：下钻、追踪、合并 |
| `flashduty-change` | 变更事件追踪与部署频率趋势 |
| `flashduty-oncall` | 值班查询：当前值班人、排班详情 |
| `flashduty-channel` | 协作空间与升级规则查询 |
| `flashduty-statuspage` | 状态页管理以及从 Atlassian 迁移到 Flashduty |
| `flashduty-insight` | 分析：MTTA/MTTR、降噪率、通知趋势 |
| `flashduty-admin` | 团队/成员查询与审计日志搜索 |
| `flashduty-template` | 通知模板验证与预览 |

---

## 快速开始

### 1. 认证

```bash
flashduty login
```

系统会提示输入 Flashduty APP Key。获取方式：登录 [Flashduty 控制台](https://console.flashcat.cloud)，进入 **账户设置 > APP Key**。

也可以通过环境变量设置：

```bash
export FLASHDUTY_APP_KEY=your_app_key
```

### 2. 使用

```bash
# 列出最近的故障
flashduty incident list

# 查看故障详情
flashduty incident get <incident_id>

# 列出团队成员
flashduty member list

# 查看协作空间
flashduty channel list
```

---

## 认证方式

CLI 按以下优先级解析凭证（优先级从高到低）：

1. `--app-key` 参数（隐藏参数，用于脚本）
2. `FLASHDUTY_APP_KEY` 环境变量
3. `~/.flashduty/config.yaml`（由 `flashduty login` 写入）

### 配置文件

存储在 `~/.flashduty/config.yaml`，权限为 `0600`：

```yaml
app_key: your_app_key
base_url: https://api.flashcat.cloud
```

### 配置命令

```bash
flashduty config show              # 查看当前配置（密钥已脱敏）
flashduty config set app_key KEY   # 设置 APP Key
flashduty config set base_url URL  # 覆盖 API 地址
```

---

## 全局参数

| 参数 | 说明 |
|------|------|
| `--json` | 以 JSON 格式输出 |
| `--no-trunc` | 表格输出时不截断长字段 |
| `--base-url` | 覆盖 API 地址 |

---

## 可用命令

### `incident` - 故障生命周期管理（9 个命令）

```bash
flashduty incident list [flags]        # 列出故障（默认最近 24 小时）
flashduty incident get <id> [<id2>]    # 查看故障详情（单个 ID 时显示详细视图）
flashduty incident create [flags]      # 创建故障（缺少参数时进入交互模式）
flashduty incident update <id> [flags] # 更新故障字段
flashduty incident ack <id> [<id2>]    # 认领故障
flashduty incident close <id> [<id2>]  # 关闭故障
flashduty incident timeline <id>       # 查看故障时间线
flashduty incident alerts <id>         # 查看故障告警
flashduty incident similar <id>        # 查找相似故障
```

**列表参数：**

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `--progress` | 筛选：Triggered、Processing、Closed | 全部 |
| `--severity` | 筛选：Critical、Warning、Info | 全部 |
| `--channel` | 按协作空间 ID 筛选 | - |
| `--title` | 按标题关键字搜索 | - |
| `--since` | 开始时间（时长、日期、日期时间或 Unix 时间戳） | `24h` |
| `--until` | 结束时间 | `now` |
| `--limit` | 最大结果数 | `20` |
| `--page` | 页码 | `1` |

**时间格式示例：** `5m`、`1h`、`24h`、`168h`、`2026-04-01`、`2026-04-01 10:00:00`、`1712000000`

### `change` - 变更记录查询（1 个命令）

```bash
flashduty change list [flags]    # 列出变更记录（部署、配置等）
```

支持 `--channel`、`--since`、`--until`、`--type`、`--limit`、`--page`。

### `member` - 成员查询（1 个命令）

```bash
flashduty member list [flags]    # 列出成员
```

支持 `--name`、`--email`、`--page`。

### `team` - 团队查询（1 个命令）

```bash
flashduty team list [flags]      # 列出团队及成员
```

支持 `--name`、`--page`。

### `channel` - 协作空间查询（1 个命令）

```bash
flashduty channel list [flags]   # 列出协作空间
```

支持 `--name`。

### `escalation-rule` - 分派策略查询（1 个命令）

```bash
flashduty escalation-rule list --channel <id>          # 按协作空间 ID 查询
flashduty escalation-rule list --channel-name <name>   # 按协作空间名称查询（自动解析）
```

### `field` - 自定义字段查询（1 个命令）

```bash
flashduty field list [flags]     # 列出自定义字段定义
```

支持 `--name`。

### `statuspage` - 状态页管理（4 个命令）

```bash
flashduty statuspage list [--id <ids>]                                  # 列出状态页
flashduty statuspage changes --page-id <id> --type <incident|maintenance>  # 列出活跃的变更
flashduty statuspage create-incident --page-id <id> --title <title>     # 创建状态页事件
flashduty statuspage create-timeline --page-id <id> --change <id> --message <msg>  # 添加时间线更新
```

### `template` - 通知模板管理（4 个命令）

```bash
flashduty template get-preset --channel <channel>                    # 获取预设模板代码
flashduty template validate --channel <channel> --file <path>        # 验证并预览模板
flashduty template variables [--category <category>]                 # 列出模板变量
flashduty template functions [--type custom|sprig|all]               # 列出模板函数
```

支持的通知渠道：`dingtalk`、`dingtalk_app`、`feishu`、`feishu_app`、`wecom`、`wecom_app`、`slack`、`slack_app`、`telegram`、`teams_app`、`email`、`sms`、`zoom`。

### 工具命令

```bash
flashduty login          # 交互式认证
flashduty config show    # 查看当前配置
flashduty config set     # 设置配置项
flashduty version        # 打印版本信息
flashduty completion     # 生成 Shell 自动补全（bash/zsh/fish/powershell）
```

---

## 输出格式

**表格（默认）：** 人类可读，列对齐，长字段自动截断。

```
ID           TITLE                    SEVERITY   PROGRESS     CHANNEL       CREATED
inc_abc123   DB connection timeout    Critical   Triggered    Production    2026-04-10 10:23
inc_def456   High memory usage        Warning    Processing   Staging       2026-04-10 09:15
Showing 2 results (page 1, total 2).
```

**JSON（`--json`）：** 机器可解析，完整数据，不截断。

```bash
flashduty incident list --json | jq '.[].title'
```

**不截断（`--no-trunc`）：** 表格显示完整字段内容。

---

## 开发

### 前置条件

- Go 1.24+
- golangci-lint（Makefile 自动安装）

### 构建

```bash
make build       # 构建二进制文件到 bin/flashduty
make test        # 运行测试（启用竞态检测）
make lint        # 运行代码检查
make check       # 运行所有检查（格式化、检查、测试、构建）
make help        # 显示所有可用目标
```

### 依赖

| 包 | 用途 |
|----|------|
| [flashduty-sdk](https://github.com/flashcatcloud/flashduty-sdk) | Flashduty API 客户端 |
| [cobra](https://github.com/spf13/cobra) | CLI 框架 |
| [yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3) | 配置文件解析 |
| [x/term](https://pkg.go.dev/golang.org/x/term) | 密码输入脱敏 |

---

## 许可证

本项目基于 MIT 许可证开源 - 详见 [LICENSE](LICENSE) 文件。
