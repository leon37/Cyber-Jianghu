# Cyber-Jianghu 本地部署指南

## 前置要求

### 硬件要求
- **GPU**: NVIDIA RTX 4070 Ti Super (16GB VRAM) 或更高
- **内存**: 32GB RAM 或更高
- **磁盘**: 100GB 可用空间
- **操作系统**: Windows 10/11 或 Linux (WSL2 推荐)

### 软件要求
- **Docker Desktop**: 4.25+ 或 Docker Compose 2.20+
- **Docker**: 24.0+
- **Git**: 2.30+
- **浏览器**: Chrome/Edge/Firefox（用于查看 ComfyUI Web UI）

## 快速开始

### 1. 克隆项目
```bash
git clone https://github.com/leon37/Cyber-Jianghu.git
cd Cyber-Jianghu
```

### 2. 配置环境变量
创建 `.env` 文件：

```bash
# ZhipuAI API Key（必需）
ZHIPUAI_API_KEY=your_zhipuai_api_key_here

# 可选配置
# COMFYUI_URL=http://comfyui:8188
# QDRANT_URL=http://qdrant:6333
# REDIS_URL=redis://redis:6379/0
```

### 3. 启动服务
```bash
# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f
```

### 4. 验证服务
```bash
# 检查服务状态
docker-compose ps

# 测试健康检查
curl http://localhost:8080/health

# 检查 ComfyUI（需要等待 1-2 分钟启动）
curl http://localhost:8188/system_stats
```

### 5. 访问服务
- **前端**: http://localhost:8080
- **ComfyUI**: http://localhost:8188
- **Qdrant Dashboard**: http://localhost:6334
- **Redis**: localhost:6379

---

## ComfyUI 部署详解

### Docker Compose 配置说明

`docker-compose.yml` 中 ComfyUI 服务的配置：

```yaml
comfyui:
  image: comfyanonymous/comfyanonymous_comfyui:latest
  container_name: comfyui
  restart: unless-stopped
  ports:
    - "8188:8188"
  volumes:
    - comfyui_models:/models
    - comfyui_output:/output
    - comfyui_input:/input
  environment:
      - CUDA_VISIBLE_DEVICES: "0"  # 使用第一个 GPU
```

### 卷说明
- `comfyui_models`: 存放 SDXL Turbo 模型文件
- `comfyui_output`: 生成的图像输出
- `comfyui_input`: 输入工作流（用于高级配置）

### 环境变量说明
- `CUDA_VISIBLE_DEVICES`: 指定使用的 GPU（0, 1, 2...）

---

## SDXL Turbo 模型配置

### 下载模型
ComfyUI 容器启动后，需要下载 SDXL Turbo 模型：

1. 访问 ComfyUI Web UI: http://localhost:8188
2. 进入 "Model Manager" 页面
3. 下载 SDXL Turbo 模型：
   - 推荐：`sd_xl_turbo_1.0_fp16.safetensors`
   - 源地址：CivitAI 或 HuggingFace
4. 将模型文件放入 `models` 目录

### 工作流配置

#### 图像生成工作流（简化版）

使用以下默认工作流用于 SDXL Turbo：

1. **CheckpointLoader** - 加载 SDXL Turbo 模型
2. **CLIPLoader** - 加载 CLIP 模型（`clip_l.safetensors`）
3. **EmptyLatentImage** - 创建空潜空间
   - Width: 1024
   - Height: 1024
   - Batch size: 1
4. **KSampler** - 采样器
   - Steps: 8（SDXL Turbo 需要更少步数）
   - CFG Scale: 7.0
   - Sampler: euler（或 dpmpp_2m）
   - Scheduler: normal
5. **VAEDecode** - VAE 解码
6. **CLIPTextEncode** - 正向提示词
7. **CLIPTextEncode** - 负向提示词
8. **SaveImage** - 保存图像
   - Filename prefix: `cyber_jianghu_`

#### 工作流导入方式

1. 保存以下 JSON 为 `.json` 文件：
   ```json
   {
     "3": {
       "inputs": {
         "seed": 12345,
         "steps": 8,
         "cfg": 7.0,
         "sampler_name": "euler",
         "scheduler": "normal",
         "denoise": 1,
         "model": ["4", 0],
         "positive": ["6", 0],
         "negative": ["7", 0],
         "latent_image": ["5", 0]
       },
       "class_type": "Sampler"
     },
     "4": {
       "inputs": {
         "ckpt_name": "sd_xl_turbo_1.0_fp16.safetensors"
       },
       "class_type": "CheckpointLoaderSimple"
     },
     "5": {
       "inputs": {
         "width": 1024,
         "height": 1024,
         "batch_size": 1
       },
       "class_type": "EmptyLatentImage"
     },
     "1": {
       "inputs": {
         "clip_name": "clip_l.safetensors"
       },
       "class_type": "CLIPLoader"
     },
     "6": {
       "inputs": {
         "text": "",
         "clip": ["1", 0]
       },
       "class_type": "CLIPTextEncode"
     },
     "7": {
       "inputs": {
         "text": "",
         "clip": ["1", 0]
       },
       "class_type": "CLIPTextEncode"
     },
     "8": {
       "inputs": {
         "filename_prefix": "cyber_jianghu_",
         "images": ["8", 0]
       },
       "class_type": "SaveImage"
     }
   }
   ```

2. 在 ComfyUI 中导入此工作流：
   - 进入 "Load" 页面
   - 点击 "Choose File"
   - 选择刚才保存的 `.json` 文件
   - 点击 "Load"

---

## GPU 资源优化

### 显存使用估算

SDXL Turbo + 1024x1024:
- **模型加载**: ~8GB VRAM
- **采样过程**: ~10GB VRAM
- **总计**: 需要至少 16GB VRAM

### 优化建议
1. **降低分辨率**: 对于快速生成，可使用 768x768
2. **减少采样步数**: SDXL Turbo 可降至 4-8 步
3. **使用固定种子**: 用于调试和一致性
4. **禁用额外功能**: 暂时不使用 ControlNet 或 LoRA

---

## 常见问题

### ComfyUI 无法连接
```bash
# 检查容器日志
docker logs comfyui

# 重启 ComfyUI
docker-compose restart comfyui

# 检查 GPU 可访问性
nvidia-smi
```

### 模型未加载
```bash
# 检查 models 目录
docker exec comfyui ls -la /models

# 确保模型文件名正确
# 在 ComfyUI Web UI 中手动指定模型路径
```

### 图像生成失败
```bash
# 检查服务器日志
docker logs server

# 检查 GPU 显存使用
nvidia-smi
```

---

## 停止服务

```bash
# 停止所有服务
docker-compose down

# 停止特定服务
docker-compose stop comfyui
docker-compose stop server
```

---

## 开发模式热重载

### 修改代码后重新构建
```bash
docker-compose down
docker-compose up --build -d server
```

### 查看 ComfyUI 日志
```bash
docker logs -f comfyui
```
