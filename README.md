# Gleaner (Go)

Gleaner 是一个用 Go 编写的中文网页内容处理库，提供正文抽取、自动摘要与情感分析能力，开箱即用，可作为库直接引入，也可以 CLI 或 HTTP 服务的形式独立部署。

## 功能特性

- **正文抽取**：基于行块密度分布算法，从杂乱 HTML 中还原主体内容
- **标题抽取**：在 `h1` 与 `title` 标签间智能选择最佳标题
- **发布日期抽取**：在正文附近通过正则匹配识别发布时间
- **自动摘要**：结合 TextRank 与神经网络多因子打分生成摘要
- **情感分析**：基于中文 AFINN 词典计算情感倾向

## 安装

```bash
go get github.com/soulteary/gleaner
```

默认分词后端 gojieba 依赖 CGO。若运行环境不便启用 CGO，可设置 `GLEANER_SEGMENT_BACKEND=gse` 切换到纯 Go 实现的 gse 后端。

## CLI

```bash
go run ./cmd/gleaner https://example.com/article.html 120
go run ./cmd/gleaner -html testdata/golden_article.html -size 200 -json
```

## 库 API

| 方法 | 说明 |
|------|------|
| `NewClient(opts Options)` | 创建客户端（推荐，可 `Close`） |
| `DefaultClient()` | 包级单例（**不可** `Close`） |
| `GetArticleSync(html, len, shingle)` | 从 HTML 同步抽取 |
| `GetArticleSyncWithOptions(html, ExtractOptions)` | 带 per-call 参数 |
| `GetArticle(url, len, shingle)` | 抓取 URL 并抽取 |
| `GetArticleWithContext(ctx, url, ExtractOptions)` | 带 context 取消/超时 |
| `Summarize(content, len, shingle, title)` | 仅摘要（支持比例 `0.04`） |
| `SummarizeWithOptions(content, ExtractOptions, title)` | 带 per-call min/max/filter |
| `Sentiment(content, inject)` | 情感分析 |

### Options

| 字段 | 默认 | 说明 |
|------|------|------|
| `SegmentBackend` | `gojieba` | `gojieba` 或 `gse` |
| `Shingle` | `false` | 摘要 shingle 模式 |
| `MinSummaryLen` | `150` | 比例摘要最短长度 |
| `MaxSummaryLen` | `350` | 比例摘要最长长度 |
| `Filter` | `[]` | 摘要句子过滤正则 |

### ExtractOptions（单次调用覆盖）

| 字段 | 说明 |
|------|------|
| `ExpectedLen` | 摘要期望长度（`Summarize` 支持 0–1 比例） |
| `Shingle` | 是否 shingle |
| `MinSummaryLen` / `MaxSummaryLen` | 覆盖客户端默认 |
| `Filter` | 覆盖客户端默认 |

### Article 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `content` | string | 正文 |
| `title` | string | 标题 |
| `pubdate` | string | 发布日期 |
| `summary` | string | 摘要 |
| `sentiment` | int | AFINN 情感总分 |

## HTTP 服务

```bash
go run ./cmd/gleaner-server
```

```bash
# 从 URL 抽取
curl -X POST -d '{"url":"https://example.com/article.html","size":100,"smooth":false}' \
  http://localhost:3824

# 从 HTML 抽取
curl -X POST -d '{"html":"<html>...</html>","size":120}' http://localhost:3824

# 健康检查
curl http://localhost:3824/healthz
```

### 错误响应

失败时返回 HTTP 4xx/5xx 与 JSON：`{"error":"..."}`（不再静默返回 `{}`）。

### 环境变量

| 变量 | 默认 | 说明 |
|------|------|------|
| `GLEANER_HOST` | `0.0.0.0` | 监听地址 |
| `GLEANER_PORT` | `3824` | 监听端口 |
| `GLEANER_SEGMENT_BACKEND` | `gojieba` | 分词后端 `gojieba`/`gse` |
| `GLEANER_BLOCK_PRIVATE_URLS` | `true`（服务端） | 拒绝抓取内网/localhost URL |

## Docker

```bash
# gojieba（CGO，默认）
docker build -t gleaner .
docker run -p 3824:3824 gleaner

# GSE 纯 Go（无 CGO，镜像更小）
docker build -f Dockerfile.gse -t gleaner:gse .
docker run -p 3824:3824 gleaner:gse
```

### 安全

HTTP 服务默认开启 SSRF 防护（`GLEANER_BLOCK_PRIVATE_URLS=true`），拒绝抓取 localhost、私有 IP 及 `.local` 域名。库客户端默认 `BlockPrivateURLs=false`；通过 `OptionsFromEnv()` 创建的服务端客户端默认开启。

## 测试

```bash
go test ./...
go vet ./...
go test -bench=. -benchmem ./internal/extractor ./... -run=^$  # BenchmarkGetBlocks 等
```

集成测试：`testdata/golden_article.html`、`testdata/golden_large_article.html`（科技类新闻样例：国产 AI 芯片发布）。

## 算法说明

| 模块 | 算法 |
|------|------|
| 正文抽取 | 行块密度分布（深度 5，rune 计分） |
| 摘要 | TextRank + 神经网络 POS 特征打分 |
| 情感 | AFINN-zh_cn + jieba 分词 + D/U 保护词 |

## 致谢与差异

本项目基于 [ahkimkoo/arex](https://github.com/ahkimkoo/arex) 的 Node.js 实现改写，在此向原作者致谢。

由于分词后端及实现细节的不同，Go 版与 Node 版在以下方面可能存在差异：

| 项目 | 说明 |
|------|------|
| 摘要句序 | gojieba 与 nodejieba 分词差异导致 TextRank 排序可能不同 |
| 情感分数 | 可能相差数个分值 |
| 用户词典 | gojieba 不加载 AFINN 单列词表 |
| `GetArticleSync` | 不做比例长度换算；`Summarize` 支持 `0.04` |

> 关于词典：原版 `user-words.txt` 为 AFINN 情感词表（单列），nodejieba 可直接加载；gojieba 加载该列表会导致词性标注异常，因此 gojieba 后端使用内置词典，情感分析仍通过 AFINN + `D_U_protection` 处理。

## 许可证

Apache v2
