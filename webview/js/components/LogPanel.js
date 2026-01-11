/* =========================================
   全局日志面板组件 (Global Log Panel Component)
   ========================================= */

import { $ } from '../utils/dom.js';

/**
 * 全局日志面板类
 * 提供可收缩的悬浮窗日志显示功能
 */
export class LogPanel {
    
    /**
     * 构造函数
     */
    constructor() {
        this.isExpanded = true;
        this.isMinimized = false;
        this.createPanel();
        this.setupEventListeners();
    }

    /**
     * 创建日志面板
     */
    createPanel() {
        // 如果面板已存在，先移除
        const existingPanel = $('#logPanel');
        if (existingPanel) {
            existingPanel.remove();
        }

        // 创建日志面板HTML结构
        const panel = document.createElement('div');
        panel.id = 'logPanel';
        panel.className = 'log-panel expanded';
        panel.innerHTML = `
            <div class="log-panel-header">
                <span class="log-panel-title">📋 系统日志</span>
                <div class="log-panel-controls">
                    <button class="log-btn" id="logClearBtn" title="清空日志">🗑️</button>
                    <button class="log-btn" id="logToggleBtn" title="收缩/展开">⬇️</button>
                </div>
            </div>
            <div class="log-panel-content">
                <div class="log-container" id="logContainer"></div>
            </div>
        `;

        document.body.appendChild(panel);
        this.container = $('#logContainer');
    }

    /**
     * 设置事件监听器
     */
    setupEventListeners() {
        $('#logClearBtn')?.addEventListener('click', () => this.clear());
        $('#logToggleBtn')?.addEventListener('click', () => this.toggle());
    }

    /**
     * 记录日志
     * @param {string} text - 日志文本
     * @param {string} type - 日志类型 (info, error, success)
     */
    log(text, type = 'info') {
        if (!this.container) return;

        const timestamp = new Date().toLocaleTimeString();
        const prefix = type === 'error' ? '❌ 错误: ' : type === 'success' ? '✅ 成功: ' : '';

        const logEntry = document.createElement('div');
        logEntry.className = `log-entry ${type}`;
        logEntry.innerHTML = `[${timestamp}] ${prefix}${this.escapeHtml(text)}`;

        this.container.appendChild(logEntry);
        this.container.scrollTop = this.container.scrollHeight;

        // 如果是最小化状态，显示新消息提示
        if (this.isMinimized) {
            this.showNewMessageIndicator();
        }
    }

    /**
     * 记录信息日志
     * @param {string} text - 日志文本
     */
    info(text) {
        this.log(text, 'info');
    }

    /**
     * 记录错误日志
     * @param {string} text - 日志文本
     */
    error(text) {
        this.log(text, 'error');
    }

    /**
     * 记录成功日志
     * @param {string} text - 日志文本
     */
    success(text) {
        this.log(text, 'success');
    }

    /**
     * 清空日志
     */
    clear() {
        if (this.container) {
            this.container.innerHTML = '';
            this.hideNewMessageIndicator();
        }
    }

    /**
     * 切换收缩/展开状态
     */
    toggle() {
        const panel = $('#logPanel');
        if (this.isExpanded) {
            panel.classList.remove('expanded');
            panel.classList.add('collapsed');
            this.isExpanded = false;
        } else {
            panel.classList.remove('collapsed');
            panel.classList.add('expanded');
            this.isExpanded = true;
        }
    }



    /**
     * HTML转义
     * @param {string} text - 需要转义的文本
     * @returns {string} 转义后的文本
     */
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}