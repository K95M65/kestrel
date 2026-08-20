import assert from "node:assert/strict";
import { mainFacePhoto } from "./facePhoto.ts";

assert.equal(mainFacePhoto(undefined), undefined);
assert.equal(mainFacePhoto([]), undefined);
assert.equal(mainFacePhoto(["a.jpg"]), "a.jpg");
assert.equal(mainFacePhoto(["100.jpg", "200.jpg", "300.jpg"]), "300.jpg");

console.log("facePhoto: ok");
