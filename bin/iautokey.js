#!/usr/bin/env node
const { spawnSync } = require("child_process");
const path = require("path");

const home = require("os").homedir();
const binary = path.join(home, ".local", "bin", "iautokey");
const result = spawnSync(binary, process.argv.slice(2), { stdio: "inherit" });
process.exit(result.status ?? 1);
