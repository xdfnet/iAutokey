#!/usr/bin/env node
"use strict";

const { spawnSync } = require("child_process");
const fs = require("fs");
const os = require("os");
const path = require("path");

if (process.platform !== "darwin") {
  console.error("iautokey: macOS only");
  process.exit(1);
}

const home = os.homedir();
const root = path.resolve(__dirname, "..");
const binDir = path.join(home, ".local", "bin");
const configDir = path.join(home, ".config", "iautokey");
const binaryPath = path.join(binDir, "iautokey");
const buildPath = path.join(root, "build", "iautokey");
const plistPath = path.join(home, "Library", "LaunchAgents", "com.user.iautokey.plist");

function run(cmd, args, opts = {}) {
  const r = spawnSync(cmd, args, { cwd: root, stdio: opts.stdio || "inherit", encoding: "utf8" });
  if (r.error) throw r.error;
  if (r.status !== 0 && !opts.allowFailure) throw new Error(`${cmd} ${args.join(" ")} failed`);
  return r;
}

function ensureDir(d) { fs.mkdirSync(d, { recursive: true }); }

function main() {
  // 查找 VERSION 优先顺序：环境变量 > package.json > git describe
  const pkg = JSON.parse(fs.readFileSync(path.join(root, "package.json"), "utf8"));
  const version = pkg.version;

  console.log(`正在编译 iautokey v${version}...`);
  ensureDir(path.dirname(buildPath));
  run("go", ["build", "-ldflags=-s -w -X main.version=" + version, "-o", buildPath, "."]);

  // 安装二进制
  ensureDir(binDir);
  ensureDir(configDir);
  fs.copyFileSync(buildPath, binaryPath);
  fs.chmodSync(binaryPath, 0o755);

  // 自签（避免丢失辅助功能权限）
  spawnSync("codesign", ["-s", "-", "-f", binaryPath], { stdio: "pipe" });

  // 首次安装创建示例配置
  const configPath = path.join(configDir, "config.json");
  if (!fs.existsSync(configPath)) {
    const example = {
      autoEnter: { enabled: true, key: "right_command", delayMs: 600 }
    };
    fs.writeFileSync(configPath, JSON.stringify(example, null, 2) + "\n");
    console.log("示例配置已创建:", configPath);
  }

  // 安装 LaunchAgent
  const plistSrc = path.join(root, "configs", "com.user.iautokey.plist");
  if (fs.existsSync(plistSrc)) {
    fs.copyFileSync(plistSrc, plistPath);
  }
  spawnSync("launchctl", ["unload", plistPath], { stdio: "ignore" });
  spawnSync("launchctl", ["load", "-w", plistPath], { stdio: "ignore" });

  // 检查是否需要辅助功能权限
  const status = run(binaryPath, ["status"], { stdio: "pipe", allowFailure: true });
  if (status.status !== 0 || status.stdout.includes("未运行")) {
    console.log(`\n⚠️  请在 系统设置→隐私与安全性→辅助功能 中添加 iautokey：
  路径: ${binaryPath}
  添加后执行: iautokey restart`);
  } else {
    console.log("\niautokey 安装成功！");
  }
}

try {
  main();
} catch (err) {
  console.error(`iautokey 安装失败: ${err.message}`);
  process.exit(1);
}
