# Cyber-Jianghu 基础设施文档

## 目录结构

```
Cyber-Jianghu/
├── server/                        # Go 后端服务
│   ├── cmd/
│   │   └── main.go             # 服务入口
│   ├── internal/
│   │   ├── adapters/           # 直播平台适配器
│   │   ├── config/             # 配置管理
│   │   ├── engine/             # AI 引擎
│   │   ├── generators/         # 资源生成（语音/图像）
│   │   ├── interfaces/         # 接口定义
│   │   ├── models/             # 数据模型
│   │   ├── prompts/            # Prompt 模板
│   │   ├── rag/               # RAG 向量检索
│   │   ├── storage/            # 存储层
│   │   └── web/               # HTTP/WebSocket
│   ├── configs/                # 配置文件
│   ├── go.mod                 # Go 模块定义
│   └── go.sum                 # 依赖锁定
├── client/                     # 前端客户端
│   ├── index.html              # 主页面
│   ├── static/
│   │   ├── css/
│   │   │   └── style.css     # 赛博武侠风格样式
│   │   └── js/
│   │       └── app.js        # 前端应用逻辑
├── PROGRESS.md                 # 开发进度
├── README.md                  # 项目说明
└── init.md                   # 初始需求文档
```

---

## 技术栈

### 后端
- **语言**: Go 1.24+
- **Web 框架**: Chi (github.com/go-chi/chi)
- **WebSocket**: Gorilla WebSocket
- **ORM**: GORM
- **HTTP Client**: 自定义 GLM-5 客户端

### AI/LLM
- **模型**: 智谱 AI GLM-5
- **Embedding**: GLM Embedding API
- **Prompt**: 模板引擎 (自定义)
- **提示词风格**: 金庸古龙江湖风格（纯正武侠，无赛博朋克元素）

### 向量数据库
- **Qdrant**: 开源向量数据库
- **客户端**: github.com/qdrant/go-client

### 缓存/队列
- **Redis**: 弹幕存储、缓存
- **客户端**: github.com/go-redis/redis/v8

### 数据库
- **MySQL**: 持久化存储
- **驱动**: gorm.io/driver/mysql

---

## 外部服务配置

### Qdrant
```yaml
# localhost:6333
# Web UI: localhost:6333/dashboard
```

### Redis
```yaml
# localhost:6379
```

### MySQL
```yaml
# localhost:3306
```

### ComfyUI (图像生成)
```yaml
# localhost:8188
```

### GPT-SoVITS (语音合成)
```yaml
# localhost:9880
```

---

## 环境变量

```bash
# 智谱 AI API Key
export ZHIPUAI_API_KEY="your_api_key"

# Qdrant API Key (可选，本地部署可留空)
export QDRANT_API_KEY=""

# Redis 密码 (可选)
export REDIS_PASSWORD=""
```

---

## 配置文件

### configs/config.yaml

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 30s
  write_timeout: 30s

database:
  mysql:
    host: "localhost"
    port: 3306
    username: "root"
    password: ""
    database: "cyber_jianghu"
    max_open_conns: 100
    max_idle_conns: 10

  redis:
    host: "localhost"
    port: 6379
    password: ""
    db: 0
    pool_size: 10

ai:
  zhipu_api_key: "${ZHIPUAI_API_KEY}"
  embedding_model: "embedding-3"
  chat_model: "glm-4"
  timeout: 120s

vector_db:
  qdrant:
    host: "localhost"
    port: 6333
    api_key: "${QDRANT_API_KEY}"
```

---

## Docker 部署

### 开发环境 Docker Compose

```yaml
version: '3.8'

services:
  qdrant:
    image: qdrant/qdrant:latest
    ports:
      - "6333:6333"
      - "6334:6334"
    volumes:
      - qdrant_data:/qdrant/storage

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

  mysql:
    image: mysql:8
    ports:
      - "3306:3306"
    environment:
      MYSQL_ROOT_PASSWORD: root
      MYSQL_DATABASE: cyber_jianghu
    volumes:
      - mysql_data:/var/lib/mysql

volumes:
  qdrant_data:
  redis_data:
  mysql_data:
```

启动命令：
```bash
docker-compose up -d
```

---

## 依赖管理

### 添加依赖
```bash
cd server
go get github.com/package/name
```

### 更新依赖
```bash
go mod tidy
```

### 清理未使用依赖
```bash
go mod tidy
```

---

## 编译与运行

### 编译
```bash
cd server
go build ./cmd/main.go
```

### 运行
```bash
# 使用配置文件
./main -config configs/config.yaml

# 或使用环境变量
export ZHIPUAI_API_KEY="xxx"
./main
```

### 开发模式（热重载）
```bash
# 安装 air
go install github.com/cosmtrek/air@latest

# 运行
air
```

---

## 测试

### 运行测试
```bash
go test ./...
```

### 运行带覆盖率的测试
```bash
go test -cover ./...
```

---

## 监控与日志

### 日志级别
- `DEBUG`: 详细调试信息
- `INFO`: 一般运行信息
- `WARN`: 警告信息
- `ERROR`: 错误信息

### 健康检查端点
```bash
curl http://localhost:8080/health
```

响应：
```json
{
  "status": "ok",
  "service": "cyber-jianghu"
}
```

## API 端点

### 故事管理 API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/story/create` | 创建新故事 |
| POST | `/api/v1/story/continue` | 继续故事（输入行动） |
| POST | `/api/v1/story/select` | 选择故事选项 |
| GET | `/api/v1/story/{story_id}` | 查询故事状态 |

### 直播管理 API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/live/connect` | 连接直播间 |
| POST | `/api/v1/live/disconnect` | 断开直播间 |
| GET | `/api/v1/live/status` | 查询连接状态 |
| WebSocket | `/api/v1/live/danmaku` | 弹幕实时流 |

---

## 数据库 Schema

### MySQL 表结构

#### stories 表
```sql
CREATE TABLE stories (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    current_scene TEXT,
    previous_text TEXT,
    summary TEXT,
    protagonist VARCHAR(255),
    npcs TEXT,
    genre VARCHAR(50),
    tone VARCHAR(50),
    style VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

#### memories 表
```sql
CREATE TABLE memories (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    content TEXT,
    metadata JSON,
    story_id BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_story_id (story_id)
);
```

### Redis 数据结构

```
danmaku:list         - List, 存储最近弹幕
danmaku:dedup:<key> - String, 弹幕去重
story:state:<id>    - Hash, 故事状态缓存
```

---

## 性能优化

### Redis 缓存策略
- 弹幕去重：TTL 5 分钟
- 弹幕列表：TTL 24 小时
- 故事状态：TTL 1 小时

### Embedding 缓存
- 默认 TTL: 24 小时
- 最大缓存数: 10,000 条

### 数据库连接池
- MySQL 最大连接: 100
- MySQL 空闲连接: 10
- Redis 连接池: 10

---

## 安全考虑

### API 密钥管理
- 使用环境变量存储密钥
- 不要将密钥提交到 Git
- 使用 `.env` 文件（已添加到 `.gitignore`）

### 跨域处理
- 开发环境允许所有来源
- 生产环境需限制允许的域名

### 限流策略
- API 调用限流（待实现）
- WebSocket 连接数限制（待实现）

---

## 故障排查

### 常见问题

1. **Qdrant 连接失败**
   - 检查 Qdrant 是否启动：`curl localhost:6333/`
   - 检查端口是否被占用

2. **Redis 连接失败**
   - 检查 Redis 是否启动：`redis-cli ping`
   - 检查密码配置

3. **GLM-5 API 调用失败**
   - 检查 API Key 是否正确
   - 检查网络连接
   - 查看日志中的错误信息

---

## 相关文档

- [PROGRESS.md](PROGRESS.md) - 开发进度
- [README.md](README.md) - 源码阅读指南
- [init.md](init.md) - 初始需求文档

---

## 许可证

详见 [LICENSE](LICENSE) 文件。
