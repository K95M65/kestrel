import assert from "node:assert/strict";
import { deviceReach, isLoopbackOrigin, mdnsHost, originPort } from "./deviceReach.ts";

assert.equal(mdnsHost("reachy-mini-a1b2"), "reachy-mini-a1b2.local");
assert.equal(mdnsHost("lamp-d94b"), "lamp-d94b.local");
assert.equal(mdnsHost("intern-v2-3c4d"), "intern-v2-3c4d.local");
assert.equal(mdnsHost("aa:bb:cc:dd:ee:ff"), "");
assert.equal(mdnsHost(""), "");
assert.equal(mdnsHost("lima-kestrel"), "");

assert.equal(originPort("http://10.10.2.160"), "");
assert.equal(originPort("http://10.10.2.160:8080"), ":8080");
assert.equal(originPort("http://127.0.0.1:5173"), ":5173");
assert.equal(isLoopbackOrigin("http://127.0.0.1:8080"), true);
assert.equal(isLoopbackOrigin("http://10.10.2.160"), false);

const desk = deviceReach({
  ip: "10.10.2.160",
  hostId: "reachy-mini-a1b2",
  origin: "http://10.10.2.160",
});
assert.equal(desk.primary, "http://10.10.2.160");
assert.equal(desk.lan, "http://10.10.2.160");
assert.equal(desk.mdns, "http://reachy-mini-a1b2.local");

const lima = deviceReach({
  ip: "192.168.5.2",
  hostId: "kestrel-host-0000",
  origin: "http://127.0.0.1:8080",
});
assert.equal(lima.primary, "http://192.168.5.2:8080");
assert.equal(lima.mdns, "http://kestrel-host-0000.local:8080");
assert.ok(!lima.primary.includes("127.0.0.1"));

const none = deviceReach({ origin: "http://localhost:5173" });
assert.equal(none.primary, "");

console.log("deviceReach: ok");
