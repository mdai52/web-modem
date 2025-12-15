class ModemManager {
    constructor() {
        this.apiBase = '/api/v1';
        this.wsUrl = `ws://${location.host}/ws`;
        this.ws = null;
        this.init();
    }

    init() {
        this.refreshPorts();
        this.setupWebSocket();
        this.setupSMSCounter();
    }

    // 设置短信字符计数器
    setupSMSCounter() {
        const textarea = document.getElementById('smsMessage');
        if (textarea) {
            // 创建计数器显示
            const counter = document. createElement('div');
            counter. id = 'smsCounter';
            counter.style.cssText = 'margin-top: 5px; color: #666; font-size:  12px;';
            textarea.parentNode.appendChild(counter);
            
            textarea.addEventListener('input', () => {
                this.updateSMSCounter();
            });
        }
    }

    // 更新短信字符计数
    updateSMSCounter() {
        const textarea = document.getElementById('smsMessage');
        const counter = document.getElementById('smsCounter');
        const message = textarea.value;
        
        // 检测是否包含中文或特殊字符
        const hasUnicode = /[^\x00-\x7F]/.test(message);
        
        let maxChars, parts;
        if (hasUnicode) {
            // UCS2 编码：70 字符单条，67 字符多条
            maxChars = message.length <= 70 ? 70 : 67;
            parts = Math.ceil(message.length / maxChars);
        } else {
            // GSM 7-bit:  160 字符单条，153 字符多条
            maxChars = message.length <= 160 ? 160 : 153;
            parts = Math.ceil(message.length / maxChars);
        }
        
        const encoding = hasUnicode ? 'UCS2 (中文)' : 'GSM 7-bit';
        counter.innerHTML = `
            <span>字符数: ${message.length} / ${maxChars}</span> | 
            <span>短信条数: ${parts}</span> | 
            <span>编码: ${encoding}</span>
        `;
        
        // 超长提示
        if (parts > 3) {
            counter.style. color = '#ff4444';
            counter.innerHTML += ' <strong>⚠️ 消息过长，将分为 ' + parts + ' 条发送</strong>';
        } else if (parts > 1) {
            counter. style.color = '#ff9800';
        } else {
            counter.style.color = '#666';
        }
    }

    // WebSocket 连接
    setupWebSocket() {
        this.ws = new WebSocket(this.wsUrl);

        this.ws.onopen = () => {
            this.addLog('WebSocket 连接已建立');
        };

        this.ws.onmessage = (event) => {
            this.addLog('收到:  ' + event.data);
        };

        this.ws.onerror = (error) => {
            this.addLog('WebSocket 错误: ' + error);
        };

        this.ws.onclose = () => {
            this.addLog('WebSocket 连接已断开');
            setTimeout(() => this.setupWebSocket(), 5000);
        };
    }

    // API 请求封装
    async apiRequest(endpoint, method = 'GET', body = null) {
        const options = {
            method,
            headers: {
                'Content-Type': 'application/json'
            }
        };

        if (body) {
            options.body = JSON.stringify(body);
        }

        try {
            const response = await fetch(this.apiBase + endpoint, options);
            const data = await response.json();
            
            if (! response.ok) {
                throw new Error(data.error || '请求失败');
            }
            
            return data;
        } catch (error) {
            this.showError(error.message);
            throw error;
        }
    }

    // 刷新串口列表
    async refreshPorts() {
        try {
            const ports = await this.apiRequest('/modems');
            const select = document.getElementById('portSelect');
            select.innerHTML = '<option value="">-- 选择串口 --</option>';
            
            ports.forEach(port => {
                const option = document.createElement('option');
                option.value = port.path;
                option.textContent = port.name + (port.connected ? ' ✅' : '');
                select.appendChild(option);
            });

            // 自动选择第一个已连接端口，减少“port is required”误点击
            const connected = ports.find(p => p.connected);
            if (connected) {
                select.value = connected.path;
            }
            
            this.addLog('已刷新串口列表');
        } catch (error) {
            console.error('刷新串口失败:', error);
        }
    }

    // 获取已选择端口，若无则提示
    getSelectedPort() {
        const port = document.getElementById('portSelect').value;
        if (!port) {
            this.showError('请选择可用串口');
            return null;
        }
        return port;
    }

    // 连接 Modem
    async connect() {
        const port = this.getSelectedPort();
        if (!port) return;

        try {
            // 后端已改为仅切换活动端口
            await this.apiRequest('/modem/connect', 'POST', { port });
            this.updateConnectionStatus(true);
            this.addLog(`已切换到端口 ${port}`);
        } catch (error) {
            console.error('连接失败:', error);
        }
    }

    // 断开 Modem
    async disconnect() {
        // 后端不再提供单端口断开，这里仅重置前端状态
        this.updateConnectionStatus(false);
        this.addLog('已清除前端连接状态');
    }

    // 发送 AT 命令
    async sendATCommand() {
        const command = document.getElementById('atCommand').value.trim();
        const port = this.getSelectedPort();
        if (!port) return;
        
        if (! command) {
            this.showError('请输入 AT 命令');
            return;
        }

        try {
            const result = await this.apiRequest('/modem/send', 'POST', { command, port });
            this.addToTerminal(`> ${command}`);
            this.addToTerminal(result.response);
            document.getElementById('atCommand').value = '';
        } catch (error) {
            console.error('发送命令失败:', error);
        }
    }

    // 获取 Modem 信息
    async getModemInfo() {
        try {
            const port = this.getSelectedPort();
            if (!port) return;
            const info = await this.apiRequest('/modem/info' + (port ? `?port=${encodeURIComponent(port)}` : ''));
            this.displayModemInfo(info);
        } catch (error) {
            console.error('获取信息失败:', error);
        }
    }

    // 获取信号强度
    async getSignalStrength() {
        try {
            const port = this.getSelectedPort();
            if (!port) return;
            const signal = await this.apiRequest('/modem/signal' + (port ? `?port=${encodeURIComponent(port)}` : ''));
            this.displaySignalInfo(signal);
        } catch (error) {
            console.error('获取信号强度失败:', error);
        }
    }

    // 列出短信
    async listSMS() {
        try {
            this.addLog('正在读取短信列表（PDU 模式）.. .');
            const port = this.getSelectedPort();
            if (!port) return;
            const smsList = await this.apiRequest('/modem/sms/list' + (port ? `?port=${encodeURIComponent(port)}` : ''));
            this.displaySMSList(smsList);
            this.addLog(`已读取 ${smsList.length} 条短信`);
        } catch (error) {
            console.error('获取短信列表失败:', error);
        }
    }

    // 发送短信
    async sendSMS() {
        const number = document. getElementById('smsNumber').value.trim();
        const message = document.getElementById('smsMessage').value.trim();
        const port = this.getSelectedPort();
        if (!port) return;

        if (!number || !message) {
            this.showError('请输入号码和短信内容');
            return;
        }

        try {
            this.addLog('正在发送短信（支持中文和长短信）...');
            await this.apiRequest('/modem/sms/send', 'POST', { 
                port,
                number, 
                message,
                usePDU: true 
            });
            this.showSuccess('短信发送成功！');
            document.getElementById('smsNumber').value = '';
            document.getElementById('smsMessage').value = '';
            this.updateSMSCounter();
        } catch (error) {
            console.error('发送短信失败:', error);
        }
    }

    // 显示 Modem 信息
    displayModemInfo(info) {
        const container = document.getElementById('modemInfo');
        container.innerHTML = `
            <div class="info-item">
                <span class="info-label">串口: </span>
                <span class="info-value">${info.port || '-'}</span>
            </div>
            <div class="info-item">
                <span class="info-label">制造商:</span>
                <span class="info-value">${info.manufacturer || '-'}</span>
            </div>
            <div class="info-item">
                <span class="info-label">型号:</span>
                <span class="info-value">${info.model || '-'}</span>
            </div>
            <div class="info-item">
                <span class="info-label">IMEI:</span>
                <span class="info-value">${info.imei || '-'}</span>
            </div>
            <div class="info-item">
                <span class="info-label">手机号:</span>
                <span class="info-value">${info.phoneNumber || '-'}</span>
            </div>
            <div class="info-item">
                <span class="info-label">运营商:</span>
                <span class="info-value">${info.operator || '-'}</span>
            </div>
        `;
    }

    // 显示信号信息
    displaySignalInfo(signal) {
        const container = document.getElementById('modemInfo');
        container.innerHTML = `
            <div class="info-item">
                <span class="info-label">信号强度 (RSSI):</span>
                <span class="info-value">${signal.rssi}</span>
            </div>
            <div class="info-item">
                <span class="info-label">信号质量: </span>
                <span class="info-value">${signal.quality}</span>
            </div>
            <div class="info-item">
                <span class="info-label">dBm:</span>
                <span class="info-value">${signal.dbm}</span>
            </div>
        `;
    }

    // 显示短信列表
    displaySMSList(smsList) {
        const container = document.getElementById('smsList');
        
        if (smsList.length === 0) {
            container. innerHTML = '<p>暂无短信</p>';
            return;
        }

        container. innerHTML = smsList.map(sms => `
            <div class="sms-item">
                <div class="sms-header">
                    <span class="sms-number">📱 ${sms.number}</span>
                    <span class="sms-time">🕐 ${sms.time}</span>
                </div>
                <div class="sms-message">${this.escapeHtml(sms.message)}</div>
            </div>
        `).join('');
    }

    // HTML 转义
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // 切换标签页
    switchTab(tabName, el) {
        document.querySelectorAll('.tab').forEach(tab => tab.classList.remove('active'));
        document.querySelectorAll('.tab-content').forEach(content => content.classList.remove('active'));

        if (el) el.classList.add('active');
        document.getElementById(tabName + 'Tab').classList.add('active');
    }

    // 更新连接状态
    updateConnectionStatus(connected) {
        const statusElement = document.getElementById('connectionStatus');
        const statusText = document.getElementById('statusText');
        
        if (connected) {
            statusElement. classList.add('connected');
            statusText.textContent = '已连接 (PDU)';
        } else {
            statusElement. classList.remove('connected');
            statusText.textContent = '未连接';
        }
    }

    // 添加到终端
    addToTerminal(text) {
        const terminal = document.getElementById('terminal');
        terminal.innerHTML += this.escapeHtml(text) + '\n';
        terminal.scrollTop = terminal.scrollHeight;
    }

    // 添加日志
    addLog(text) {
        const log = document.getElementById('log');
        const timestamp = new Date().toLocaleTimeString();
        log.innerHTML += `[${timestamp}] ${this.escapeHtml(text)}\n`;
        log.scrollTop = log.scrollHeight;
    }

    // 清空日志
    clearLog() {
        document.getElementById('log').innerHTML = '';
    }

    // 显示错误
    showError(message) {
        this.addLog('❌ 错误: ' + message);
        alert('错误: ' + message);
    }

    // 显示成功
    showSuccess(message) {
        this.addLog('✅ 成功:  ' + message);
    }
}

// 初始化应用
const app = new ModemManager();

// 回车发送 AT 命令
document.getElementById('atCommand')?.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') {
        app.sendATCommand();
    }
});