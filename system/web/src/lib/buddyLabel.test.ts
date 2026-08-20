import assert from "node:assert/strict";
import { buddyKind, buddyOSLabel } from "./buddyLabel.ts";

assert.equal(buddyOSLabel("macOS 14.5"), "macOS");
assert.equal(buddyOSLabel("Windows 10.0.26100"), "Windows");
assert.equal(buddyOSLabel("Linux amd64"), "Linux");
assert.equal(buddyOSLabel("Ubuntu 24.04"), "Linux");
assert.equal(buddyOSLabel(""), "Computer");
assert.equal(buddyKind([
  { id: "autonomous-buddy", kind: "buddy" },
  { id: "dance", kind: "robot-app" },
  { id: "autonomous-buddy-linux", kind: "buddy" },
]).length, 2);

console.log("buddyLabel: ok");
