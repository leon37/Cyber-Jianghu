// Cyber Jianghu - Frontend Application

class CyberJianghu {
    constructor() {
        this.ws = null;
        this.isConnected = false;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.currentMode = 'demo'; // 'demo' or 'live'

        this.init();
    }

    init() {
        this.bindEvents();
        this.startDemoMode();
        this.log('系统初始化完成', 'success');
    }

    bindEvents() {
        // Mode selection
        document.querySelectorAll('.mode-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                this.switchMode(e.target.dataset.mode);
            });
        });

        // Demo form
        document.getElementById('demo-form').addEventListener('submit', (e) => {
            e.preventDefault();
            this.handleDemoAction();
        });

        // Connect form (live mode)
        document.getElementById('connect-form').addEventListener('submit', (e) => {
            e.preventDefault();
            this.handleConnect();
        });

        // Disconnect button
        document.getElementById('disconnect-btn').addEventListener('click', () => {
            this.handleDisconnect();
        });

        // Handle window close
        window.addEventListener('beforeunload', () => {
            if (this.ws) {
                this.ws.close();
            }
        });
    }

    switchMode(mode) {
        this.currentMode = mode;
        this.log(`切换到 ${mode === 'demo' ? '演示' : '直播'} 模式`, 'info');

        // Update button states
        document.querySelectorAll('.mode-btn').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.mode === mode);
        });

        // Show/hide forms
        if (mode === 'demo') {
            document.getElementById('demo-form').style.display = 'flex';
            document.getElementById('connect-form').style.display = 'none';
            this.startDemoMode();
        } else {
            document.getElementById('demo-form').style.display = 'none';
            document.getElementById('connect-form').style.display = 'flex';
            this.startLiveMode();
        }
    }

    async startDemoMode() {
        this.log('进入演示模式', 'success');

        // Try to create a new story
        try {
            const response = await fetch('/api/v1/story/create', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    genre: '武侠',
                    tone: '赛博',
                    style: '说书风',
                    protagonist: '玩家'
                })
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }

            const data = await response.json();
            this.log('故事创建成功', 'success');
            this.displayInitialStory(data);
        } catch (error) {
            this.log(`故事创建失败: ${error.message}`, 'error');
            // Display welcome message
            this.updateStory('欢迎来到赛博江湖！');
            this.updateOptions([
                { id: '1', text: '走进客栈' },
                { id: '2', text: '拜访武林盟主' },
                { id: '3', text: '查看系统状态' }
            ]);
        }
    }

    displayInitialStory(data) {
        if (data.story) {
            this.updateStory(data.story.text || data.story.content);
            this.updateOptions(data.story.options || []);
        }
    }

    startLiveMode() {
        this.log('进入直播模式', 'info');
        this.connectWebSocket();
    }

    connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/api/v1/live/danmaku`;

        try {
            this.ws = new WebSocket(wsUrl);
            this.setupWebSocketHandlers();
        } catch (error) {
            this.log(`WebSocket 连接失败: ${error.message}`, 'error');
            this.updateConnectionStatus(false);
        }
    }

    setupWebSocketHandlers() {
        this.ws.onopen = () => {
            this.log('WebSocket 连接成功', 'success');
            this.updateConnectionStatus(true);
            this.reconnectAttempts = 0;
        };

        this.ws.onclose = () => {
            this.log('WebSocket 连接关闭', 'info');
            this.updateConnectionStatus(false);

            // Attempt reconnection
            if (this.reconnectAttempts < this.maxReconnectAttempts && this.currentMode === 'live') {
                this.reconnectAttempts++;
                const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
                this.log(`${delay/1000} 秒后尝试重连... (第 ${this.reconnectAttempts} 次)`, 'info');
                setTimeout(() => this.connectWebSocket(), delay);
            }
        };

        this.ws.onerror = (error) => {
            this.log(`WebSocket 错误: ${error}`, 'error');
        };

        this.ws.onmessage = (event) => {
            this.handleMessage(JSON.parse(event.data));
        };
    }

    handleMessage(data) {
        switch (data.type) {
            case 'connected':
                this.handleConnected(data);
                break;
            case 'danmaku':
                this.handleDanmaku(data);
                break;
            case 'story':
                this.handleStory(data);
                break;
            case 'image':
                this.handleImage(data);
                break;
            case 'audio':
                this.handleAudio(data);
                break;
            default:
                this.log(`未知消息类型: ${data.type}`, 'info');
        }
    }

    handleConnected(data) {
        this.log(`已连接到弹幕流，客户端 ID: ${data.id}`, 'success');
    }

    handleDanmaku(data) {
        this.displayDanmaku(data.username || '观众', data.content, data.msg_type);

        // Check for commands
        if (data.msg_type === 'command') {
            this.log(`收到命令: ${data.content}`, 'info');
        }
    }

    handleStory(data) {
        this.updateStory(data.text || data.content);
        this.updateOptions(data.options);
        this.log('故事已更新', 'success');
    }

    handleImage(data) {
        this.updateVisual(data.image_url || data.image);
        this.log('图像已更新', 'success');
    }

    handleAudio(data) {
        this.playAudio(data.audio_url || data.audio);
        this.log('音频播放中', 'success');
    }

    displayDanmaku(username, content, type) {
        const container = document.getElementById('danmaku-display');
        const entry = document.createElement('div');
        entry.className = 'danmaku-entry';

        let contentHtml = '';
        if (type === 'command') {
            contentHtml = `<span class="danmaku-command">${content}</span>`;
        } else {
            contentHtml = content;
        }

        entry.innerHTML = `
            <span class="danmaku-username">${username}:</span>
            <span class="danmaku-content">${contentHtml}</span>
        `;

        container.insertBefore(entry, container.firstChild);

        // Limit entries
        while (container.children.length > 100) {
            container.removeChild(container.lastChild);
        }
    }

    updateStory(text) {
        const display = document.getElementById('story-display');

        // Parse markdown if available
        if (typeof marked !== 'undefined') {
            display.innerHTML = marked.parse(text);
        } else {
            display.textContent = text;
        }
    }

    updateVisual(imageUrl) {
        const display = document.getElementById('visual-display');

        if (imageUrl) {
            display.innerHTML = `<img src="${imageUrl}" alt="场景图像" />`;
        } else {
            display.innerHTML = '<div class="placeholder">图像区域</div>';
        }
    }

    updateOptions(options) {
        const container = document.getElementById('options-display');
        container.innerHTML = '';

        if (!options || options.length === 0) {
            container.innerHTML = '<div class="loading">暂无选项</div>';
            return;
        }

        options.forEach((option, index) => {
            const btn = document.createElement('button');
            btn.className = 'option-btn';
            btn.textContent = option.text || option.description || option;
            btn.dataset.optionId = option.id || index;

            btn.addEventListener('click', () => {
                this.handleOptionSelect(option, btn);
            });

            container.appendChild(btn);
        });
    }

    async handleDemoAction() {
        const action = document.getElementById('action').value;

        if (!action) {
            alert('请输入行动');
            return;
        }

        this.log(`执行行动: ${action}`, 'info');

        // Display the action in danmaku
        this.displayDanmaku('玩家', action, 'normal');

        // Simulate story response (demo mode)
        try {
            const response = await fetch('/api/v1/story/continue', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    story_id: 'current',
                    action: action
                })
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }

            const data = await response.json();
            this.displayInitialStory(data);
        } catch (error) {
            this.log(`故事继续失败: ${error.message}`, 'error');
            // Demo fallback - generate simple responses
            this.generateDemoResponse(action);
        }
    }

    generateDemoResponse(action) {
        // Simple demo responses for testing
        const responses = [
            `你${action}，周围安静得出奇...`,
            `系统：${action} 已记录，正在生成响应...`,
            `由于这是演示模式，${action} 只触发了一个示例响应。`
        ];

        const randomResponse = responses[Math.floor(Math.random() * responses.length)];
        this.updateStory(randomResponse);
        this.updateOptions([
            { id: '1', text: '继续探索' },
            { id: '2', text: '查看装备' },
            { id: '3', text: '返回' }
        ]);

        // Clear action input
        document.getElementById('action').value = '';
    }

    async handleOptionSelect(option, btn) {
        // Update button state
        document.querySelectorAll('.option-btn').forEach(b => b.classList.remove('selected'));
        btn.classList.add('selected');

        this.log(`选择选项: ${option.text || option.description || option}`, 'info');

        // Send selection to server (via HTMX or fetch)
        try {
            const response = await fetch('/api/v1/story/select', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    story_id: 'current',
                    option_id: option.id || btn.dataset.optionId,
                    choice_text: option.text || option.description || option
                })
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }

            const data = await response.json();
            this.log('选项提交成功', 'success');

            // Display the story response
            if (data.story) {
                this.displayInitialStory(data);
            }
        } catch (error) {
            this.log(`选项提交失败: ${error.message}`, 'error');

            // Demo fallback
            this.generateDemoResponse(option.text || option.description || option);
        }
    }

    playAudio(audioUrl) {
        const audio = document.getElementById('audio-player');
        if (audioUrl) {
            audio.src = audioUrl;
            audio.play().catch(error => {
                this.log(`音频播放失败: ${error.message}`, 'error');
            });
        }
    }

    async handleConnect() {
        const platform = document.getElementById('platform').value;
        const roomId = document.getElementById('room_id').value;
        const cookie = document.getElementById('cookie').value;

        if (!roomId) {
            alert('请输入房间号');
            return;
        }

        this.log(`正在连接 ${platform} 直播间 ${roomId}...`, 'info');

        try {
            const response = await fetch('/api/v1/live/connect', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    platform: platform,
                    room_id: roomId,
                    cookie: cookie
                })
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }

            const data = await response.json();
            this.log('直播连接成功', 'success');

            // Update UI state
            document.getElementById('connect-btn').disabled = true;
            document.getElementById('disconnect-btn').disabled = false;
            document.getElementById('platform').disabled = true;
            document.getElementById('room_id').disabled = true;

        } catch (error) {
            this.log(`直播连接失败: ${error.message}`, 'error');
            alert(`连接失败: ${error.message}`);
        }
    }

    async handleDisconnect() {
        this.log('正在断开直播连接...', 'info');

        try {
            const response = await fetch('/api/v1/live/disconnect', {
                method: 'POST'
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }

            this.log('直播已断开', 'success');

            // Update UI state
            document.getElementById('connect-btn').disabled = false;
            document.getElementById('disconnect-btn').disabled = true;
            document.getElementById('platform').disabled = false;
            document.getElementById('room_id').disabled = false;

            // Reset story display
            document.getElementById('story-display').innerHTML = '<div class="loading">等待故事开始...</div>';

        } catch (error) {
            this.log(`断开连接失败: ${error.message}`, 'error');
        }
    }

    updateConnectionStatus(connected) {
        this.isConnected = connected;
        const indicator = document.getElementById('live-status');
        const dot = indicator.querySelector('.status-dot');
        const text = indicator.querySelector('.status-text');

        if (connected) {
            dot.classList.add('connected');
            text.textContent = '已连接';
        } else {
            dot.classList.remove('connected');
            text.textContent = '未连接';
        }
    }

    log(message, type = 'info') {
        const container = document.getElementById('debug-log');
        const entry = document.createElement('div');
        entry.className = `log-entry ${type}`;

        const timestamp = new Date().toLocaleTimeString();
        entry.textContent = `[${timestamp}] ${message}`;

        container.insertBefore(entry, container.firstChild);

        // Limit entries
        while (container.children.length > 50) {
            container.removeChild(container.lastChild);
        }

        // Also log to console
        console.log(`[${type.toUpperCase()}] ${message}`);
    }
}

// Initialize application when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    window.app = new CyberJianghu();
});
