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

## Phase 2: 直播接入 🔄 进行中

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

### 进行中任务
- [ ] WebSocket 服务搭建 (`internal/web/handlers.go`)
  - [ ] 添加 WebSocket 端点 `/api/v1/live/danmaku/stream`
  - [ ] 实现弹幕实时推送
  - [ ] 客户端连接管理

- [ ] 弹幕广播器
  - [ ] 创建 DanmakuHub 管理器
  - [ ] 实现多客户端广播

### 待完成任务
- [ ] 集成 Bilibili 适配器到 HTTP 服务
- [ ] 实现连接/断开 API (`/api/v1/live/connect`, `/api/v1/live/disconnect`)
- [ ] 弹幕存储到 Redis
- [ ] 测试 Bilibili 直播间连接

### 已创建文件
```
server/internal/
├── adapters/
│   ├── bilibili.go                    # Bilibili 适配器（已完成）
│   ├── douyin.go                     # Douyin 适配器（骨架）
│   └── parser.go                     # 弹幕解析器
```

---

## Phase 3: 剧情引擎 + RAG 集成 ⏸️ 待开始

### 待完成任务
- [ ] 集成 GLM-5 API (`internal/engine/glm5_client.go`)
- [ ] 实现 Prompt 模板引擎 (`internal/prompts/story_template.go`)
- [ ] 实现剧情状态管理 (`internal/models/story.go`)
- [ ] 集成 Qdrant 向量数据库 (`internal/rag/qdrant_client.go`)
- [ ] 实现 Embedding 服务 (`internal/rag/embedding.go`)
- [ ] 实现历史记忆存储 (`internal/rag/memory_store.go`)
- [ ] 集成 RAG 检索模板 (`internal/prompts/rag_template.go`)
- [ ] MySQL/GORM 存档实现
- [ ] Redis 缓存实时状态

---

## Phase 4: 本地 AIGC ⏸️ 待开始

### 待完成任务
- [ ] ComfyUI 本地部署文档
- [ ] SDXL Turbo 模型配置
- [ ] ComfyUI API 客户端 (`internal/generators/comfyui_client.go`)
- [ ] 图像生成工作流创建
- [ ] LoRA 模型管理实现
- [ ] 图像缓存机制
- [ ] 显存优化

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
