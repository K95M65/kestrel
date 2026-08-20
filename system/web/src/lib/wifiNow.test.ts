import assert from "node:assert/strict";
import { formatWifiNow } from "./wifiNow.ts";

assert.equal(formatWifiNow(null), "Not on a station network.");
assert.equal(formatWifiNow({}), "Not on a station network.");
assert.equal(formatWifiNow({ ssid: "HomeNet" }), "HomeNet");
assert.equal(formatWifiNow({ ssid: "HomeNet", signal: -48, linkRate: 433 }), "HomeNet · -48 dBm · 433 Mbps");
assert.equal(formatWifiNow({ ssid: "  " }), "Not on a station network.");

console.log("wifiNow: ok");
