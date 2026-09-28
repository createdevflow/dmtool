import { describe, expect, it } from "vitest";
import { accessTokenExpiresAtMs } from "./access-token";

function fakeJwt(payload: object): string {
  const json = JSON.stringify(payload);
  const b64 = Buffer.from(json).toString("base64url");
  return `aaa.${b64}.sig`;
}

describe("accessTokenExpiresAtMs", () => {
  it("reads exp in milliseconds", () => {
    const exp = 1_700_000_000;
    expect(accessTokenExpiresAtMs(fakeJwt({ exp }))).toBe(exp * 1000);
  });

  it("returns null for garbage", () => {
    expect(accessTokenExpiresAtMs("not-a-jwt")).toBeNull();
    expect(accessTokenExpiresAtMs(fakeJwt({}))).toBeNull();
  });
});
