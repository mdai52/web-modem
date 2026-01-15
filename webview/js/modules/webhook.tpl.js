/**
 * 预设模板定义
 * 包含模板数据及显示名称
 */
export const PRESET_TEMPLATES = {
    generic: {
        name: "通用格式",
        template: {
            event: "sms_received",
            data: {
                content: "{{content}}",
                send_number: "{{send_number}}",
                receive_number: "{{receive_number}}",
                receive_time: "{{receive_time}}",
                sms_ids: "{{sms_ids}}",
                direction: "{{direction}}"
            },
            timestamp: "{{receive_time}}"
        }
    },
    wechat_work: {
        name: "企业微信机器人",
        template: {
            msgtype: "text",
            text: {
                content: "收到短信\n发件人: {{send_number}}\n收件人: {{receive_number}}\n内容: {{content}}\n时间: {{receive_time}}"
            }
        }
    },
    feige: {
        name: "飞鸽传书",
        template: {
            title: "新短信通知",
            content: "发件人: {{send_number}}\n收件人: {{receive_number}}\n内容: {{content}}\n时间: {{receive_time}}",
            timestamp: "{{receive_time}}"
        }
    },
    dingtalk: {
        name: "钉钉机器人",
        template: {
            msgtype: "text",
            text: {
                content: "【短信通知】\n发件人: {{send_number}}\n收件人: {{receive_number}}\n内容: {{content}}\n时间: {{receive_time}}"
            }
        }
    },
    feishu: {
        name: "飞书机器人",
        template: {
            msg_type: "text",
            content: {
                text: "【短信通知】\n发件人: {{send_number}}\n收件人: {{receive_number}}\n内容: {{content}}\n时间: {{receive_time}}"
            }
        }
    },
    discord: {
        name: "Discord",
        template: {
            content: "📱 **收到新短信**",
            embeds: [
                {
                    title: "短信详情",
                    color: 5814783,
                    fields: [
                        {
                            name: "发件人",
                            value: "{{send_number}}",
                            inline: true
                        },
                        {
                            name: "收件人",
                            value: "{{receive_number}}",
                            inline: true
                        },
                        {
                            name: "内容",
                            value: "{{content}}"
                        },
                        {
                            name: "时间",
                            value: "{{receive_time}}",
                            inline: true
                        }
                    ],
                    timestamp: "{{receive_time}}"
                }
            ]
        }
    },
    slack: {
        name: "Slack",
        template: {
            text: "📱 收到新短信",
            blocks: [
                {
                    type: "header",
                    text: {
                        type: "plain_text",
                        text: "短信通知"
                    }
                },
                {
                    type: "section",
                    fields: [
                        {
                            type: "mrkdwn",
                            text: "*发件人:*\n{{send_number}}"
                        },
                        {
                            type: "mrkdwn",
                            text: "*收件人:*\n{{receive_number}}"
                        }
                    ]
                },
                {
                    type: "section",
                    text: {
                        type: "mrkdwn",
                        text: "*内容:*\n{{content}}"
                    }
                },
                {
                    type: "section",
                    text: {
                        type: "mrkdwn",
                        text: "*时间:* {{receive_time}}"
                    }
                }
            ]
        }
    },
    telegram: {
        name: "Telegram Bot",
        template: {
            chat_id: "",
            text: "📱 *新短信通知*\n\n发件人: `{{send_number}}`\n收件人: `{{receive_number}}`\n内容: {{content}}\n时间: {{receive_time}}",
            parse_mode: "Markdown"
        }
    }
};