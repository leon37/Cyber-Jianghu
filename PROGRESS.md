# Cyber-Jianghu 项目进度

**项目名称**: 赛博江湖 - AI原生互动直播游戏

**开始时间**: 2026-01-27
**最后更新**: 2026-02-12

---

## 项目概述

AI 原生互动直播游戏，核心玩法是观众弹幕指令驱动剧情发展，实时生成图像、语音和文本并推流至直播平台。

### 核心特点
- 本地推理（RTX 4070 Ti Super 16G）：ComfyUI + SDXL Turbo + GPT-SoVITS
- Vibe Coding 开发模式：GLM-5 作为产品/逻辑大脑
- 弹幕驱动剧情：观众发送弹幕指令/投票来推动故事发展
- 实时视觉生成：根据剧情内容生成场景图并推流
- 实时语音合成：根据剧情文本生成语音旁白
- 金庸古龙江湖风格：纯正的武侠世界观

---

## 开发进度

### Phase 1: 基础搭建 ✅ 已完成
- [x] Go 项目目录结构搭建
- [x] 基础 HTTP 服务器（Chi 框架 + 自定义处理器）
- [x] 配置管理系统（YAML）
- [x] 日志系统
- [x] 项目初始化脚本

### Phase 2: 直播接入 ✅ 已完成
- [x] WebSocket 实时通信（Gorilla）
- [x] 弹幕广播器（DanmakuHub）
- [x] Bilibili 直播平台适配器
- [x] Douyin 直播平台适配器（骨架）
- [x] 弹幕命令解析器（`/action <param>`, `/vote <option>`）

### Phase 3: 故事管理 API ✅ 已完成
- [x] 故事创建 API（`POST /api/v1/story/create`）
- [x] 故事继续 API（`POST /api/v1/story/continue`）
- [x] 选项选择 API（`POST /api/v1/story/select`）
- [x] 故事状态查询 API（`GET /api/v1/story/{story_id}`）
- [x] 金庸古龙江湖风格提示词模板
- [x] 故事状态机管理（内存存储）
- [x] 选项解析逻辑（支持 A.B.C. 和数字格式）

### Phase 4: 前端开发 ✅ 已完成
- [x] 主页面 HTML（index.html）
- [x] 赛博武侠风格 CSS（static/css/style.css）
- [x] 前端应用逻辑（static/js/app.js）
- [x] WebSocket 连接管理
- [x] 弹幕实时显示
- [x] 故事文本显示
- [x] 选项按钮交互
- [x] 调试模式 vs 直播模式切换
- [x] 防止重复创建故事的标志位

### Phase 5: AI 引擎层 ✅ 部分完成
- [x] **GLM-5 客户端**（`internal/engine/glm5_client.go`）
  - ChatCompletion API 封装
  - Embedding API 封装
  - 错误处理和重试机制
- [x] **Qdrant 向量数据库客户端**（`internal/rag/qdrant_client.go`）
  - 简化端点创建和管理
- [x] **故事引擎核心**（`internal/engine/story_engine.go`）
  - StoryState 状态机
  - StoryResponse 响应结构
  - GenerateStorySegment 剧情生成
  - ApplyOption 选项应用
  - parseOptionsFromResponse 选项解析
- [x] **Prompt 模板引擎**（`internal/prompts/story_template.go`）
  - 故事续写模板（金庸古龙江湖风格）
  - 场景描述模板
  - NPC 对话模板
  - 决策总结模板
- [x] **RAG 向量检索**（`internal/rag/`）
  - Qdrant 集合
  - Embedding 服务
  - 向量存储
  - 相关记忆检索
  - 相关决策检索

### Phase 6: 前端表现层 🚧 进行中

- [x] 主页面结构完善
- [x] 弹幕实时显示优化
- [x] 图像显示区域
- [x] 选项按钮样式
- [x] 故事文本样式（支持 Markdown）
- [x] 模式切换界面
- [ ] 音频播放器集成

### Phase 7: 直播平台集成 ⏳️ 待开始
- [ ] Bilibili 直播房间连接优化
- [ ] Douyin 直播房间连接优化
- [ ] 心跳保活机制
- [ ] 弹幕去重和过滤
- [ ] 高并发弹幕场景处理

### Phase 8: 图像生成 ⏳️ 待开始
- [ ] ComfyUI 集成（本地 SDXL Turbo）
- [ ] 图像缓存机制
- [ ] 请求队列管理
- [ ] 图像生成 API 端点
- [ ] 场景描述提示词优化

### Phase 9: 语音合成 ⏳️ 待开始
- [x] GPT-SoVITS 语音合成集成
- [ ] 语音缓存机制
- [ ] 音频生成 API 端点
- [ ] 音频格式转换和缓存
- [ ] 默认音色管理

---

## 技术架构

### 后端技术栈
```
语言: Go 1.23+
Web 框架: Chi (github.com/go-chi/chi/v5)
WebSocket: Gorilla WebSocket
ORM: GORM (计划用于 MySQL 持久化）
向量数据库: Qdrant (本地推理 + 存储)
AI 模型: 智谱 AI GLM-5 (Chat + Embedding)
语音合成: GPT-SoVITS (本地)
图像生成: Stable Diffusion XL Turbo (本地)
```

### 前端技术栈
- 语言: JavaScript
- 样式: 原生 CSS
- 依赖: marked.js (Markdown 渲染)
- WebSocket: 原生 API

### 核心组件交互
```
观众弹幕/指令
    ↓
    ↓
↓
DanmakuHub (WebSocket 广播中心)
    ├─→ multiple WebSocket 客户端连接
    ↓
StoryEngine (故事引擎)
    ↓
    ├─→ GLM-5Client (Chat API)
    ├─→ QdrantClient (RAG 向量检索)
    └─→ MemoryStore (记忆存储)
    └─→ PromptEngine (提示词渲染)
    ↓
    └─→ 生成剧情 + 选项
    ↓
StoryResponse (HTTP 响应)
    ├─→ Text (剧情文本)
    ├─→ Options (选项按钮)
    └─→ VisualPrompt (图像提示词)
    └─→ AudioPrompt (语音提示词)
        ↓
    ↓ (WebSocket 广播)
Frontend Clients
```
```

---

## 当前问题与已知问题

### 选项解析问题 ⚠️ 已修复
- **问题**: 选项提取逻辑不完善，导致显示默认选项而非 LLM 生成的选项
- **根本原因**: LLM 可能未按指定格式生成选项（A. B. C. 或数字格式）
- **临时解决方案**: 当未找到选项时提供默认选项（继续前进、观察周围、询问NPC）
- **需优化**: 改进正则表达式，支持更多选项格式

### 故事状态管理 ⚠️ 已修复
- **问题**: 初始化状态包含乱码字符
- **根本原因**: 中文编码问题（UTF-8 处理不当）
- **临时解决方案**: 简化提示词模板，去除现代科技词汇
- **状态**: 需验证实际运行时 LLM 生成的效果

### WebSocket 连接 ⚠️ 已实现
- **问题**: 部署模式未实现
- **状态**: 基础连接已完成
- **待完成**: 添加认证机制、断线重连、心跳保活

### MySQL 持久化 ⚠️ 待完成
- **状态**: Stub 实现（内存存储）
- **原因**: 开发阶段使用内存存储更快速
- **待完成**: 添加 GORM 集成、定义模型和迁移脚本

---

## 下一步计划

### 优先级 P0 - 核心功能完善
1. **修复选项解析逻辑**
   - 改进正则表达式，支持更多选项格式
   - 添加更多测试用例
   - 实现更智能的选项提取（NLP 或 LLM 辅助）

2. **完善 WebSocket 功能**
   - 添加认证机制（JWT）
   - 实现断线重连
   - 添加心跳保活机制
   - 改进错误处理和日志记录

3. **前端体验优化**
   - 添加加载动画和过渡效果
   - 优化故事文本显示（Markdown 渲染）
   - 添加音效和视觉反馈
   - 改进错误提示信息

4. **性能监控**
   - 添加请求/响应日志
   - 添加性能指标追踪
   - 添加健康检查端点

### 优先级 P1 - 图像生成
1. **ComfyUI 集成**
   - 完成 SDXL Turbo 模型部署
   - 实现图像生成 API 端点
   - 实现图像缓存机制
   - 添加图像质量优化（提示词调整）

2. **图像缓存策略**
   - 定义缓存键生成规则
   - 实现缓存查询逻辑
   - 实现缓存失效和淘汰策略
   - 添加缓存命中率统计

### 优先级 P2 - 语音合成
1. **GPT-SoVITS 集成**
   - 完成本地语音合成服务部署
- 实现语音生成 API 端点
- 实现语音缓存机制
- 添加音频格式转换（wav, mp3）
- 实现默认音色和音色切换

### 优先级 P2 - 数据库集成
1. **MySQL 持久化**
   - 定义数据模型和关系
- - 编写 GORM 集成
- - 创建迁移脚本
- - 实现 Story 和 Memory 模型的持久化

2. **Redis 优化**
   - 定义缓存策略
- - 实现弹幕数据缓存
- - 实现故事状态缓存

### 优先级 P3 - 直播功能完善
1. **Bilibili 直播优化**
   - 测试高并发弹幕场景
- 优化弹幕去重和过滤
- 添加弹幕速率限制
- 实现礼物特效和感谢机制

2. **Douyin 直播优化**
- 完成弹幕解析器实现
- 测试高并发弹幕场景
- 优化弹幕去重和过滤

---

## 环境要求

### 运行环境
- Go 1.23+ 或更高
- Node.js 20+（可选，用于图像生成）
- Python 3.10+（可选，用于语音合成）
- 8GB+ RAM
- 20GB+ 磁盘空间

### 开发环境
- VS Code（推荐）或 GoLand
- Git

### 外部服务依赖
- Qdrant: localhost:6333（本地开发可空）
- ComfyUI: localhost:8188（本地开发可空）
- GPT-SoVITS: 本地服务或云端 API
- SDXL Turbo: 本地部署或云端 API

---

## 已知限制

### 性能
- 当前无并发控制，需要限制同时在线的 WebSocket 连接数
- LLM API 有请求频率限制
- 图像生成和语音合成是 CPU 密集操作

### 功能限制
- 直播平台适配器当前只支持弹幕接收
- 未实现点赞、礼物、关注等交互功能
- 图像生成依赖本地模型，需要额外硬件资源

### 安全
- API 密钥管理需使用环境变量
- WebSocket 连接未实现认证
- SQL 注入未实现参数化

---

## 提交与协作
- 项目使用 Git 进行版本控制
- Pull Request 欢迎
- Issue 报告欢迎

## 代码统计

```
总行数: ~2000+
Go 代码行数: ~1500+
前端代码行数: ~500+
测试覆盖率: ~20% (故事 API + 基础功能)
```

---

**最后更新**: 2026-02-12 - 添加金庸古龙江湖风格提示词模板
