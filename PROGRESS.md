# Cyber-Jianghu 项目进度

**项目名称**: 赛博江湖 - AI原生互动直播游戏
**开始时间**: 2026-01-27
**最后更新**: 2026-02-12

---

## 项目概述

AI 原生互动直播游戏，核心玩法是观众弹幕指令驱动剧情发展，实时生成图像+语音+文本并推流至直播平台。

### 核心特点
- 本地推理（RTX 4070 Ti Super 16G）：ComfyUI + SDXL Turbo + GPT-SoVITS
- Vibe Coding 开发模式：GLM-5 作为产品/逻辑大脑，Claude Code 代码落地
- 实时性要求：弹幕到反馈 < 5秒
- RAG 支持：使用 Qdrant 存储和检索历史剧情/角色决策

---

## Phase 1: 基础搭建 ✅ 已完成

### 已完成任务
- [x] 创建 Go 项目目录结构
  - [x] `internal/adapters/` - 直播平台适配器
  - [x] `internal/engine/` - 剧情引擎
  - [x] `internal/generators/` - 资源生成器
  - [x] `internal/rag/` - RAG 向量检索
  - [x] `internal/interfaces/` - 接口定义
  - [x] `internal/models/` - 数据模型
  - [x] `internal/prompts/` - Prompt 模板
  - [x] `internal/queue/` - 任务队列
  - [x] `internal/resilience/` - 容错机制
  - [x] `internal/storage/` - 数据存储
  - [x] `internal/web/` - HTTP/WebSocket 服务
  - [x] `pkg/` - 公共包
  - [x] `configs/` - 配置文件

- [x] 更新 go.mod 依赖
  - [x] github.com/go-chi/chi (HTTP 路由)
  - [x] github.com/golang-jwt/jwt/v4 (JWT 认证)
  - [x] github.com/gorilla/websocket (WebSocket)
  - [x] github.com/go-redis/redis/v8 (Redis 客户端)
  - [x] github.com/sashabaranov/go-openai (GLM-5 兼容 API 客户端)
  - [x] gopkg.in/yaml.v2 (YAML 配置)
  - [x] gorm.io/gorm + gorm.io/driver/mysql (ORM + MySQL 驱动)
  - [x] github.com/qdrant/go-client (Qdrant 客户端)

- [x] 搭建基础 HTTP 服务器 (`cmd/main.go`)
- [x] 实现健康检查接口 (`GET /health`)
- [x] 实现配置加载 (`internal/config/config.go`)
- [x] 实现存储抽象层
  - [x] MySQL 存储实现 (`internal/storage/mysql.go`)
  - [x] Redis 存储实现 (`internal/storage/redis.go`)

### 已创建文件
```
server/
├── cmd/main.go                          # 服务入口
├── go.mod / go.sum                      # 依赖管理
├── configs/config.yaml                  # 配置文件
├── internal/
│   ├── config/config.go                # 配置加载
│   ├── storage/mysql.go                # MySQL 实现
│   ├── storage/redis.go                # Redis 实现
│   ├── web/handlers.go                # HTTP 处理器
│   ├── interfaces/
│   │   ├── live_adapter.go            # 直播适配器接口
│   │   ├── story_engine.go            # 剧情引擎接口
│   │   ├── asset_gen.go               # 资源生成接口
│   │   └── vector_store.go            # 向量存储接口
│   └── models/
│       ├── story.go                    # 故事模型
│       ├── lora_registry.go           # LoRA 模型
│       └── memory.go                  # 记忆模型
```

### 验证状态
- ✅ HTTP 服务正常启动（端口 8080）
- ✅ 健康检查接口响应正常
- ✅ 所有依赖编译通过
- ✅ 目录结构完整

---

## Phase 2: 直播接入 ✅ 已完成

### 已完成任务
- [x] 定义 LiveAdapter 接口 (`internal/interfaces/live_adapter.go`)
- [x] 实现 Bilibili 直播适配器 (`internal/adapters/bilibili.go`)
  - [x] WebSocket 连接
  - [x] 弹幕消息订阅
  - [x] 心跳保活
  - [x] 消息解析
- [x] 创建 Douyin 适配器骨架 (`internal/adapters/douyin.go`)
- [x] 实现弹幕解析器 (`internal/adapters/parser.go`)
  - [x] 命令格式解析: `/action <param>`
  - [x] 投票格式解析: `/vote <optionID>`
- [x] 创建 DanmakuHub 弹幕广播器 (`internal/web/hub.go`)
  - [x] 多客户端连接管理
  - [x] 弹幕实时广播
  - [x] 客户端注册/注销
  - [x] 心跳保活机制
- [x] WebSocket 服务搭建 (`internal/web/handlers.go`)
  - [x] 添加 WebSocket 端点 `/api/v1/live/danmaku/stream`
  - [x] 实现弹幕实时推送
  - [x] 客户端连接管理
- [x] 实现直播连接/断开 API
  - [x] 创建 LiveService 管理器 (`internal/web/live_service.go`)
  - [x] `POST /api/v1/live/connect` 连接直播间
  - [x] `POST /api/v1/live/disconnect` 断开直播间
  - [x] `GET /api/v1/live/status` 查询连接状态
- [x] 弹幕存储到 Redis (`internal/storage/redis.go`)
  - [x] 弹幕数据存储（List 结构）
  - [x] 去重机制（Dedup key）
  - [x] 获取最近弹幕 API
  - [x] 弹幕计数和清理

### 待完成任务
- [ ] 测试 Bilibili 直播间连接
- [ ] 性能测试（高并发弹幕场景）

### 已创建文件
```
server/internal/
├── adapters/
│   ├── bilibili.go                    # Bilibili 适配器（已完成）
│   ├── douyin.go                     # Douyin 适配器（骨架）
│   └── parser.go                     # 弹幕解析器
└── web/
    ├── hub.go                        # 弹幕广播器
    ├── live_service.go               # 直播服务管理器
    └── handlers.go                   # HTTP/WebSocket 处理器（已更新）
```

### API 端点
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/health` | 健康检查 |
| POST | `/api/v1/live/connect` | 连接直播间 (Body: `{"platform":"bilibili","room_id":"xxx","cookie":"xxx"}`) |
| POST | `/api/v1/live/disconnect` | 断开直播间 |
| GET | `/api/v1/live/status` | 查询连接状态 |
| WebSocket | `/api/v1/live/danmaku` | 弹幕实时流 |

---

## Phase 3: 剧情引擎 + RAG 集成 ✅ 已完成

### 已完成任务
- [x] 集成 GLM-5 API (`internal/engine/glm5_client.go`)
  - [x] ChatCompletion 接口
  - [x] Embedding 接口
  - [x] 错误处理和重试机制
  - [x] 请求限流控制
- [x] 实现 Prompt 模板引擎 (`internal/prompts/story_template.go`)
  - [x] 基础剧情 Prompt 模板
  - [x] 变量插值支持
  - [x] 图像生成 Prompt 模板
  - [x] NPC 对话模板
  - [x] 决策总结模板
- [x] 集成 Qdrant 向量数据库 (`internal/rag/qdrant_client.go`)
  - [x] 创建 Collection
  - [x] 存储 Embedding
  - [x] 向量检索（相似度搜索）
  - [x] 元数据过滤
  - [x] 连接池管理
- [x] 实现 Embedding 服务 (`internal/rag/embedding.go`)
  - [x] 调用 GLM Embedding API
  - [x] 本地缓存机制
  - [x] 批量 Embedding 支持
  - [x] 向量归一化
  - [x] 余弦相似度计算
- [x] 实现历史记忆存储 (`internal/rag/memory_store.go`)
  - [x] 记忆写入到 Qdrant
  - [x] 相关记忆检索
  - [x] 记忆元数据管理
  - [x] 记忆类型过滤
- [x] 实现剧情引擎核心逻辑 (`internal/engine/story_engine.go`)
  - [x] 状态机管理
  - [x] 玩家决策处理
  - [x] 剧情分支生成
  - [x] 选项解析
  - [x] RAG 检索集成
- [x] 集成 RAG 检索到剧情引擎
  - [x] 基于玩家输入检索相关记忆
  - [x] 检索相似决策
  - [x] 注入到 Prompt 上下文
  - [x] 检索结果排序和过滤

### 待完成任务
- [ ] 实现 MySQL/GORM 存档实现
- [ ] Redis 缓存实时状态
- [ ] 故事 API 端点集成

### 已创建文件
```
server/internal/
├── engine/
│   ├── glm5_client.go               # GLM-5 API 客户端
│   └── story_engine.go             # 剧情引擎核心逻辑
├── prompts/
│   └── story_template.go           # Prompt 模板引擎
└── rag/
    ├── qdrant_client.go            # Qdrant 向量数据库客户端
    ├── embedding.go                # Embedding 服务
    └── memory_store.go            # 历史记忆存储服务
```

### 核心数据结构

**StoryState** - 故事状态
```go
type StoryState struct {
    CurrentNode   string         // 当前节点
    CurrentScene  string         // 当前场景
    PreviousText  string         // 之前文本
    Summary       string         // 故事摘要
    Protagonist   string         // 主角
    NPCs          string         // NPC 列表
    Genre          string         // 类型
    Tone           string         // 语调
    Style          string         // 风格
    Options        []StoryOption // 可选选项
    Custom         map[string]interface{} // 自定义数据
}
```

**Memory** - 记忆存储
```go
type Memory struct {
    ID        string                 // 唯一 ID
    Type      MemoryType            // 类型: player_action, story_state, npc, decision
    Content   string                 // 内容
    Timestamp int64                  // 时间戳
    StoryID   string                 // 故事 ID
    Metadata  map[string]interface{} // 元数据
    Vector    []float64               // 向量（内部使用）
}
```

**StoryResponse** - 剧情生成响应
```go
type StoryResponse struct {
    Text           string         // 生成的文本
    Scene          string         // 场景描述
    Options        []StoryOption // 选项
    VisualPrompt   string         // 图像生成提示
    AudioPrompt    string         // 语音生成提示
    RelatedMemories []Memory      // 相关记忆
}
```

### 数据流

```
玩家弹幕/选择
    ↓
StoryEngine.ApplyOption()
    ↓
MemoryStore.SearchRelatedMemories()  ← RAG 检索
    ↓
TemplateEngine.Render()  ← 构建 Prompt
    ↓
GLM5Client.Chat()  ← 调用大模型
    ↓
StoryEngine.parseOptionsFromResponse()  ← 解析选项
    ↓
MemoryStore.StoreDecision()  ← 存储决策
    ↓
返回 StoryResponse (含 VisualPrompt)
```

---

## Phase 4: 本地 AIGC ✅ 已完成

### 已完成任务
- [x] ComfyUI API 客户端 (`internal/generators/comfyui_client.go`)
  - [x] 连接到本地 ComfyUI 服务
  - [x] SDXL Turbo 工作流构建
  - [x] 异步任务处理
  - [x] 轮询生成结果
  - [x] 队列状态查询
- [x] LoRA 模型管理 (`internal/generators/lora_manager.go`)
  - [x] 模型注册和加载
  - [x] 模型元数据管理
  - [x] 动态模型切换
  - [x] 人物模型支持
- [x] 图像缓存机制 (`internal/generators/image_cache.go`)
  - [x] 生成历史缓存
  - [x] 按提示词匹配缓存
  - [x] 缓存过期策略
  - [x] 缓存命中率统计
  - [x] LRU 淘汰策略

### 待完成任务
- [ ] ComfyUI 本地部署文档
- [ ] SDXL Turbo 模型配置指南
- [ ] 显存优化（动态 Batch Size）
- [ ] 图像生成 API 端点集成

### 已创建文件
```
server/internal/
└── generators/
    ├── comfyui_client.go           # ComfyUI API 客户端
    ├── lora_manager.go            # LoRA 模型管理
    └── image_cache.go             # 图像缓存机制
```

### 核心数据结构

**GenerateOptions** - 图像生成选项
```go
type GenerateOptions struct {
    Prompt        string   // 提示词
    NegativePrompt string   // 负面提示词
    Width         int      // 图像宽度
    Height        int      // 图像高度
    Steps         int      // 推理步数
    CFGScale      float64  // CFG scale
    Seed          int      // 随机种子
    Model         string   // 模型名称
    Lora          string   // LoRA 模型
    LoraStrength  float64  // LoRA 强度
    SamplerName   string   // 采样器
    Scheduler     string   // 调度器
}
```

**CacheEntry** - 缓存条目
```go
type CacheEntry struct {
    Key           string                 // 缓存键（MD5）
    FilePath      string                 // 文件路径
    Prompt        string                 // 原始提示词
    Options       *GenerateOptions        // 生成选项
    CreatedAt     time.Time              // 创建时间
    LastAccessed  time.Time              // 最后访问时间
    AccessCount   int                    // 访问次数
    Hits         int                    // 命中次数
    Metadata      map[string]interface{} // 元数据
}
```

### 工作流程

```
1. 收到图像生成请求（带 Prompt 和 Options）
    ↓
2. 生成缓存键（MD5 of Prompt + Options）
    ↓
3. 检查缓存
    ├─ 命中 → 返回缓存的图像
    └─ 未命中 → 继续
    ↓
4. 构建 ComfyUI 工作流
    ↓
5. 提交到 ComfyUI 队列
    ↓
6. 轮询生成结果
    ↓
7. 获取生成的图像
    ↓
8. 存储到缓存
    ↓
9. 返回图像数据
```

### 缓存策略

- **键生成**: MD5(Prompt + Width + Height + Steps + CFG + Model + LoRA)
- **过期时间**: 24 小时（可配置）
- **淘汰策略**: LRU（最近最少使用）
- **最大条目数**: 1000 条（可配置）
- **命中率统计**: 实时追踪

---

## Phase 5: 语音合成 ⏸️ 待开始

### 待完成任务
- [ ] GPT-SoVITS 本地部署
- [ ] 训练数据准备（说书人音色）
- [ ] 音色模型训练
- [ ] GPT-SoVITS API 客户端 (`internal/generators/sovits_client.go`)
- [ ] 音频格式转换
- [ ] 音频缓存机制

---

## Phase 6: 前端表现 ⏸️ 待开始

### 待完成任务
- [ ] HTML 模板结构设计
- [ ] Go Template 渲染实现
- [ ] HTMX 前端集成
- [ ] WebSocket SSE 推送
- [ ] UI 元素（血条、打赏特效）
- [ ] CSS 赛博武侠风格设计
- [ ] OBS 浏览器源配置指南

### 待创建文件
```
client/
├── index.html
├── static/
│   └── css/
│       └── style.css
└── js/
    └── htmx.min.js
```

---

## Phase 7: 集成测试 ⏸️ 待开始

### 待完成任务
- [ ] 端到端流程测试
- [ ] 负载测试（弹幕并发）
- [ ] 显存压力测试
- [ ] 容错机制验证
- [ ] 降级策略测试
- [ ] RAG 检索准确性测试

---

## Phase 8: 部署上线 ⏸️ 待开始

### 待完成任务
- [ ] Docker 镜像构建
- [ ] Docker Compose 编排
- [ ] 监控日志集成
- [ ] 告警配置
- [ ] 操作文档编写
- [ ] 应急预案制定

---

## 环境配置

### 开发环境
- **操作系统**: Windows 10 Pro
- **Go 版本**: 1.16
- **项目目录**: `D:\Cyber-Jianghu`

### 外部服务（待配置）
- [ ] Qdrant (localhost:6333)
- [ ] Redis (localhost:6379)
- [ ] MySQL (localhost:3306)
- [ ] ComfyUI (localhost:8188)
- [ ] GPT-SoVITS (localhost:9880)

### 环境变量
```bash
ZHIPUAI_API_KEY="YOUR_ZHIPUAI_API_KEY"
QDRANT_API_KEY="YOUR_QDRANT_API_KEY"
```

---

## 已知问题

### Go 版本兼容性
- 当前使用 Go 1.16，部分新特性不可用
- 计划后续升级到 Go 1.21+

### 依赖版本
- 由于 Go 1.16 限制，使用了一些旧版本依赖包

---

## 下一步行动

1. 完成 Phase 2: WebSocket 服务搭建
2. 集成测试 Bilibili 直播间连接
3. 开始 Phase 3: 剧情引擎实现

---

## 备注

- 项目采用 Vibe Coding 开发模式
- 技术栈参考 `init.md` 文档
- 接口定义参考 `server/internal/interfaces/` 目录
