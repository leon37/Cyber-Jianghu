# Cyber-Jianghu 源码阅读指南

## 项目概述

赛博江湖是一个 AI 原生互动直播游戏，核心玩法是通过观众弹幕指令驱动剧情发展，实时生成图像、语音和文本并推流至直播平台。

**核心特色**：
- 金庸古龙江湖风格：纯正的武侠世界观，无赛博朋克元素
- LLM 驱动剧情：GLM-5 模型根据用户输入生成剧情和选项
- 双模式运行：演示模式（单机测试）+ 直播模式（弹幕驱动）

---

## 架构图

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        Cyber-Jianghu 架构图                           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────────┐     │
│  │                    📦 应用入口层                         │     │
│  │  cmd/main.go                                               │     │
│  │  ↓                                                         │     │
│  └─────────────────────────────────────────────────────────────────┘     │
│                            ↓                                       │
│  ┌─────────────────────────────────────────────────────────────────┐     │
│  │                    🌐 HTTP/WebSocket 层                 │     │
│  │  web/handlers.go  ← 入口处理器                            │     │
│  │  web/hub.go         ← 弹幕广播器                          │     │
│  │  web/live_service.go ← 直播服务管理器                       │     │
│  └─────────────────────────────────────────────────────────────────┘     │
│         ↓                    ↓                    ↓                      │
│  ┌─────────────────────────────────────────────────────────────────┐     │
│  │                    🔌 直播平台适配器                     │     │
│  │  adapters/bilibili.go      ← B站适配器 (已完成)              │     │
│  │  adapters/douyin.go       ← 抖音适配器 (骨架)               │     │
│  │  adapters/parser.go       ← 弹幕解析器                     │     │
│  │  ↓                      ↑                                  │     │
│  └─────────────────────────────────────────────────────────────────┘     │
│                            ↓                                       │
│  ┌─────────────────────────────────────────────────────────────────┐     │
│  │                    🧠 AI 引擎层                        │     │
│  │  engine/glm5_client.go    ← GLM-5 API 客户端             │     │
│  │  prompts/story_template.go ← Prompt 模板引擎                 │     │
│  │  rag/embedding.go        ← Embedding 服务                   │     │
│  │  rag/qdrant_client.go    ← 向量数据库客户端                 │     │
│  └─────────────────────────────────────────────────────────────────┘     │
│         ↓                    ↓                                       │
│  ┌─────────────────────────────────────────────────────────────────┐     │
│  │                    💾 存储层                           │     │
│  │  storage/mysql.go        ← MySQL 持久化存储               │     │
│  │  storage/redis.go        ← Redis 缓存/弹幕存储              │     │
│  └─────────────────────────────────────────────────────────────────┘     │
│                            ↓                                       │
│  ┌─────────────────────────────────────────────────────────────────┐     │
│  │                    🧩 核心数据模型                       │     │
│  │  models/story.go          ← 故事模型                       │     │
│  │  models/memory.go         ← 记忆模型                       │     │
│  │  models/lora_registry.go  ← LoRA 模型注册                  │     │
│  │  interfaces/*.go           ← 接口定义                      │     │
│  └─────────────────────────────────────────────────────────────────┘     │
│                                                                     │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 模块阅读优先级排序

### 🥇 优先级 1: 核心入口与配置

**目标**: 理解项目如何启动，以及配置项有哪些

**文件列表**:
```
📁 入口层 (cmd)
└── cmd/main.go                    ← 📍 起点！理解服务启动流程

📁 配置层 (config)
└── config/config.go                ← 配置加载逻辑
```

**阅读重点**:
- 服务如何初始化（HTTP 服务器、存储连接）
- 配置项有哪些（端口、数据库、API 密钥等）
- 优雅关闭如何处理

**阅读方法**:
1. 先看 `main.go` 的 `main()` 函数，了解启动流程
2. 再看 `config.go`，了解配置结构
3. 理解各组件的初始化顺序

**预计阅读时间**: 10-15 分钟

---

### 🥈 优先级 2: 接口定义与数据模型

**目标**: 理解系统能做什么，以及数据如何表示

**文件列表**:
```
📁 接口定义 (interfaces)
├── live_adapter.go               ← 直播平台统一接口
├── story_engine.go               ← 剧情引擎接口
├── vector_store.go               ← 向量存储接口
└── asset_gen.go                 ← 资源生成接口

📁 数据模型 (models)
├── story.go                     ← 故事状态模型
├── memory.go                    ← 记忆模型
└── lora_registry.go             ← LoRA 模型注册
```

**阅读重点**:
- 核心抽象设计（为什么需要这些接口）
- 数据结构如何表示业务概念
- 接口之间的依赖关系

**阅读方法**:
1. 先看 `interfaces/`，理解系统设计意图
2. 再看 `models/`，理解数据表示
3. 思考接口与模型的关系

**预计阅读时间**: 20-30 分钟

---

### 🥉 优先级 3: HTTP/WebSocket 层

**目标**: 理解系统如何对外提供服务

**文件列表**:
```
📁 Web 层 (web)
├── handlers.go                   ← HTTP 端点入口
├── hub.go                       ← WebSocket 连接管理
└── live_service.go              ← 直播服务生命周期
```

**阅读重点**:
- REST API 设计有哪些端点
- WebSocket 如何处理连接和消息
- 如何将弹幕广播给多个客户端

**阅读方法**:
1. 先看 `handlers.go`，了解 API 路由
2. 再看 `hub.go`，理解 WebSocket 广播机制
3. 最后看 `live_service.go`，理解直播服务逻辑

**预计阅读时间**: 30-40 分钟

---

### 4️⃣ 优先级 4: 直播平台适配器

**目标**: 理解如何接入不同直播平台

**文件列表**:
```
📁 适配器层 (adapters)
├── parser.go                     ← 弹幕命令解析
├── bilibili.go                   ← B站实现（完整）
└── douyin.go                    ← 抖音实现（骨架）
```

**阅读重点**:
- 适配器模式的应用
- 弹幕协议的解析
- 心跳保活机制

**阅读方法**:
1. 先看 `parser.go`，理解命令解析
2. 重点看 `bilibili.go`，理解完整实现
3. 对比 `douyin.go`，理解待开发部分

**预计阅读时间**: 40-50 分钟

---

### 5️⃣ 优先级 5: AI 引擎层

**目标**: 理解如何调用大模型和 RAG 检索

**文件列表**:
```
📁 AI 引擎 (engine)
└── story_engine.go              ← 故事引擎核心（新增）
└── glm5_client.go                ← GLM-5 API 调用

📁 Prompt 模板 (prompts)
└── story_template.go             ← 模板渲染逻辑（含金庸风格提示词）

📁 资源生成 (generators)
└── gptsovits_client.go          ← GPT-SoVITS 语音合成（新增）
└── audio_cache.go               ← 音频缓存（新增）
└── voice_registry.go            ← 音色注册（新增）

📁 RAG 向量检索 (rag)
├── embedding.go                 ← 文本向量化
└── qdrant_client.go             ← 向量存储与检索
```

**阅读重点**:
- API 调用方式和错误处理
- Prompt 工程技巧
- RAG 检索流程

**阅读方法**:
1. 先看 `glm5_client.go`，了解 API 调用
2. 再看 `story_template.go`，理解 Prompt 模板
3. 最后看 `rag/` 目录，理解向量检索

**预计阅读时间**: 40-60 分钟

---

### 6️⃣ 优先级 6: 存储层

**目标**: 理解数据如何持久化和缓存

**文件列表**:
```
📁 存储层 (storage)
├── mysql.go                     ← MySQL 操作
└── redis.go                     ← Redis 操作
```

**阅读重点**:
- 数据如何持久化到 MySQL
- Redis 缓存策略
- 弹幕队列设计

**阅读方法**:
1. 看 `redis.go`，重点理解弹幕存储
2. 看 `mysql.go`，了解持久化逻辑

**预计阅读时间**: 20-30 分钟

---

## 数据流调用关系

### 最小可运行链路（MVP）

```
用户弹幕 → adapters/bilibili.go → web/live_service.go → web/hub.go
                                                          ↓
                                              播放端/OBS (WebSocket)
```

### AI 生成链路

```
web/handlers.go → prompts/story_template.go → engine/glm5_client.go
                                                      ↓
                                          rag/embedding.go
                                                      ↓
                                          rag/qdrant_client.go
                                                      ↓
                                          storage/redis/mysql.go
```

---

## 快速阅读路径

### 路径 1: 理解直播弹幕流程（最快上手）

```
1. cmd/main.go (启动)
2. web/handlers.go (API 入口)
3. web/live_service.go (服务管理)
4. adapters/bilibili.go (弹幕接收)
5. web/hub.go (弹幕广播)
```

### 路径 2: 理解 AI 生成流程

```
1. prompts/story_template.go (Prompt 构建)
2. engine/glm5_client.go (API 调用)
3. rag/embedding.go (向量化)
4. rag/qdrant_client.go (检索)
```

### 路径 3: 理解完整数据流

```
1. interfaces/*.go (接口定义)
2. models/*.go (数据模型)
3. storage/*.go (存储实现)
4. 然后回到路径 1 或 2
```

---

## 各模块详细说明

### cmd/main.go

**职责**: 应用入口，初始化所有组件

**关键函数**:
- `main()` - 服务启动流程

**阅读提示**:
- 关注组件初始化顺序
- 关注优雅关闭处理

---

### web/handlers.go

**职责**: HTTP 请求处理

**关键处理器**:
- `ConnectLive()` - 连接直播间
- `DisconnectLive()` - 断开直播间
- `GetLiveStatus()` - 获取状态
- `GetDanmakuStream()` - WebSocket 弹幕流

**阅读提示**:
- 关注路由设计
- 关注错误处理

---

### web/hub.go

**职责**: WebSocket 连接管理和消息广播

**关键结构**:
- `DanmakuHub` - 弹幕广播中心
- `Client` - 客户端连接

**阅读提示**:
- 关注并发安全
- 关注广播机制

---

### web/live_service.go

**职责**: 直播服务生命周期管理

**关键方法**:
- `Connect()` - 连接直播平台
- `Disconnect()` - 断开连接
- `forwardDanmaku()` - 转发弹幕

**阅读提示**:
- 关注适配器模式
- 关注错误处理

---

### adapters/bilibili.go

**职责**: B站直播平台适配器

**关键方法**:
- `Connect()` - 建立 WebSocket 连接
- `readMessages()` - 读取消息
- `parseDanmaku()` - 解析弹幕

**阅读提示**:
- 关注协议解析
- 关注心跳机制

---

### adapters/parser.go

**职责**: 弹幕命令解析

**关键函数**:
- `Parse()` - 解析弹幕
- `IsActionCommand()` - 判断是否为命令

**阅读提示**:
- 关注正则表达式
- 关注命令格式定义

---

### engine/glm5_client.go

**职责**: GLM-5 API 客户端

**关键方法**:
- `Chat()` - 聊天完成
- `CreateEmbedding()` - 生成向量

**阅读提示**:
- 关注错误重试
- 关注超时处理

---

### prompts/story_template.go

**职责**: Prompt 模板引擎

**关键方法**:
- `Render()` - 渲染模板
- `BuildStoryContext()` - 构建上下文

**阅读提示**:
- 关注模板语法
- 关注变量替换

---

### rag/embedding.go

**职责**: 文本 Embedding 服务

**关键方法**:
- `Embed()` - 单文本向量化
- `EmbedBatch()` - 批量向量化

**阅读提示**:
- 关注缓存策略
- 关注批量优化

---

### rag/qdrant_client.go

**职责**: Qdrant 向量数据库客户端

**关键方法**:
- `CreateCollection()` - 创建集合
- `InsertPoints()` - 插入向量
- `Search()` - 相似度搜索

**阅读提示**:
- 关注连接管理
- 关注检索参数

---

### storage/redis.go

**职责**: Redis 操作封装

**关键方法**:
- `StoreDanmaku()` - 存储弹幕
- `GetRecentDanmaku()` - 获取最近弹幕

**阅读提示**:
- 关注数据结构设计
- 关注过期策略

---

### storage/mysql.go

**职责**: MySQL 操作封装

**阅读提示**:
- 关注 GORM 使用
- 关注连接池配置

---

## API 端点参考

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/health` | 健康检查 |
| POST | `/api/v1/live/connect` | 连接直播间 |
| POST | `/api/v1/live/disconnect` | 断开直播间 |
| GET | `/api/v1/live/status` | 查询连接状态 |
| WebSocket | `/api/v1/live/danmaku` | 弹幕实时流 |
| POST | `/api/v1/story/create` | 创建新故事 |
| POST | `/api/v1/story/continue` | 继续故事（输入行动） |
| POST | `/api/v1/story/select` | 选择故事选项 |
| GET | `/api/v1/story/{story_id}` | 查询故事状态 |

---

## 开发进度

详见 `PROGRESS.md` 文件。

---

## 贡献指南

阅读代码后如有疑问或建议，欢迎提出 Issue 或 Pull Request。
