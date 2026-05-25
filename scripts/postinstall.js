#!/usr/bin/env node
const { spawnSync } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");

if (process.platform !== "darwin") {
  console.error("iautokey: macOS only");
  process.exit(1);
}

const root = path.resolve(__dirname, "..");
const dist = path.join(root, "dist");
const buildPath = path.join(dist, "iautokey");
const home = require("node:os").homedir();
const binDir = path.join(home, ".local", "bin");

const pkg = JSON.parse(fs.readFileSync(path.join(root, "package.json"), "utf8"));

console.log(`正在编译 iautokey v${pkg.version}...`);
fs.mkdirSync(dist, { recursive: true });
const r = spawnSync("go", ["build", "-ldflags", `-s -w -X main.version=${pkg.version}`, "-o", buildPath, "."], {
  cwd: root, stdio: "inherit",
});
if (r.status !== 0) process.exit(r.status);

fs.mkdirSync(binDir, { recursive: true });
fs.copyFileSync(buildPath, path.join(binDir, "iautokey"));
fs.chmodSync(path.join(binDir, "iautokey"), 0o755);

console.log("\n✅ iautokey 已安装");
console.log("首次使用运行: iautokey setup");
console.log("升级后重启:   iautokey restart");
