import assert from "node:assert/strict";
import { runtimeKind, runtimeSwitchWarning } from "./agentRuntime.ts";

assert.equal(runtimeKind("openclaw"), "companion");
assert.equal(runtimeKind("Hermes"), "companion");
assert.equal(runtimeKind("codex"), "coding");
assert.equal(runtimeKind("claudecode"), "coding");
assert.equal(runtimeKind("opencode"), "coding");
assert.equal(runtimeKind("picoclaw"), "telegram");
assert.equal(runtimeKind("mystery"), "unknown");

assert.equal(runtimeSwitchWarning("openclaw"), "");
assert.match(runtimeSwitchWarning("codex"), /coding CLI/i);
assert.match(runtimeSwitchWarning("claudecode"), /OpenClaw or Hermes/);
assert.match(runtimeSwitchWarning("picoclaw"), /Telegram-only/);
assert.equal(runtimeSwitchWarning("hermes"), "");

console.log("agentRuntime: ok");
