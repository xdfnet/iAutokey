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
const targetPath = path.join(binDir, "iautokey");
const DEFAULT_SIGN_ID = "4A287668E97BC130AA6D19F4D64799394CAACBAD";
const signId = process.env.IAUTOKEY_SIGN_ID || DEFAULT_SIGN_ID;

const pkg = JSON.parse(fs.readFileSync(path.join(root, "package.json"), "utf8"));

console.log(`正在编译 iautokey v${pkg.version}...`);
fs.mkdirSync(dist, { recursive: true });
const r = spawnSync("go", ["build", "-ldflags", `-s -w -X main.version=${pkg.version}`, "-o", buildPath, "."], {
  cwd: root, stdio: "inherit",
});
if (r.status !== 0) process.exit(r.status);

fs.mkdirSync(binDir, { recursive: true });
fs.copyFileSync(buildPath, targetPath);
fs.chmodSync(targetPath, 0o755);

const hasIdentity = spawnSync("security", ["find-identity", "-v", "-p", "codesigning"], {
  stdio: "pipe",
  encoding: "utf8",
});
const canSign = hasIdentity.status === 0 && hasIdentity.stdout.includes(signId);
if (canSign) {
  const sign = spawnSync(
    "codesign",
    ["-s", signId, "-f", "--identifier", "com.user.iautokey", targetPath],
    { stdio: "inherit" },
  );
  if (sign.status !== 0) {
    console.warn("⚠️ codesign 失败，后续可能需要重新授予辅助功能权限");
  } else {
    const verify = spawnSync("codesign", ["-dv", targetPath], { stdio: "pipe", encoding: "utf8" });
    const detail = `${verify.stdout || ""}${verify.stderr || ""}`;
    const line = detail.split("\n").find((s) => s.includes("Authority=") || s.includes("TeamIdentifier="));
    if (line) console.log(`✅ 已签名: ${line.trim()}`);
  }
} else {
  console.warn(`⚠️ 未找到签名证书 ${signId}，跳过签名；可能需要重新授予辅助功能权限`);
}

console.log("\n✅ iautokey 已安装");
console.log("首次使用运行: iautokey setup");
console.log("升级后重启:   iautokey restart");
