# iAutokey

[![npm](https://img.shields.io/npm/v/@xdfnet/iautokey)](https://npm.im/@xdfnet/iautokey)

macOS 按键事件监听工具。按住修饰键说话，松开即自动发送 Enter。

配合语音输入法（系统听写、第三方语音输入）使用，实现无障碍语音录入。

## 安装

```bash
npm i -g @xdfnet/iautokey
```

## 配置

编辑 `~/.config/iautokey/config.json`：

```json
{
  "autoEnter": {
    "enabled": true,
    "key": "right_command",
    "delayMs": 600
  }
}
```

可用按键：`right_command`、`left_command`、`right_option`、`left_option`、`right_shift`、`left_shift`、`right_control`、`left_control`、`fn`

## 使用

```bash
iautokey               # 启动守护进程（launchd 管理）
iautokey status        # 查看状态
iautokey restart       # 重启
iautokey version       # 版本号
```

首次使用需在 **系统设置 → 隐私与安全性 → 辅助功能** 中添加 `iautokey`。

## 开机自启

npm 安装后自动配置 LaunchAgent。手动管理：

```bash
launchctl load ~/Library/LaunchAgents/com.user.iautokey.plist
```

## 从 iSpeak 迁移

如果之前通过 iSpeak 使用 autoEnter 功能，配置会自动兼容读取（`~/.config/ispeak/config.json`）。

建议将配置迁移到独立路径：

```bash
cp ~/.config/ispeak/config.json ~/.config/iautokey/config.json
```

## 构建

```bash
git clone https://github.com/xdfnet/iAutokey.git
cd iAutokey
make deploy
```

## 许可

MIT
