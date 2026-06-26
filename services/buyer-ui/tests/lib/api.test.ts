import { decodeJwtUser, friendlyOrderId } from "../../lib/api";

function makeJwt(payload: Record<string, unknown>): string {
  const encoded = Buffer.from(JSON.stringify(payload)).toString("base64url");
  return `header.${encoded}.signature`;
}

describe("decodeJwtUser", () => {
  it("returns name and email from valid JWT payload", () => {
    const token = makeJwt({ full_name: "Alice Smith", email: "alice@example.com" });
    expect(decodeJwtUser(token)).toEqual({ name: "Alice Smith", email: "alice@example.com" });
  });

  it("returns null for both when JWT has fewer than 3 parts (malformed)", () => {
    expect(decodeJwtUser("onlyone")).toEqual({ name: null, email: null });
    expect(decodeJwtUser("only.two")).toEqual({ name: null, email: null });
  });

  it("returns null for both when payload is not valid base64", () => {
    const token = "header.!!!not-valid-base64!!$.signature";
    expect(decodeJwtUser(token)).toEqual({ name: null, email: null });
  });

  it("returns null for both when payload JSON lacks name/email fields", () => {
    const token = makeJwt({ sub: "user-123", role: "buyer" });
    expect(decodeJwtUser(token)).toEqual({ name: null, email: null });
  });

  it("falls back to payload.name when payload.full_name is absent", () => {
    const token = makeJwt({ name: "Bob Jones", email: "bob@example.com" });
    expect(decodeJwtUser(token)).toEqual({ name: "Bob Jones", email: "bob@example.com" });
  });

  it("handles payload.full_name taking priority over payload.name", () => {
    const token = makeJwt({ full_name: "Carol Full", name: "Carol Short", email: "carol@example.com" });
    expect(decodeJwtUser(token)).toEqual({ name: "Carol Full", email: "carol@example.com" });
  });
});

describe("friendlyOrderId", () => {
  it("returns #ZM- prefix followed by last 8 chars uppercased", () => {
    const result = friendlyOrderId("abcdef1234567890");
    expect(result).toBe("#ZM-34567890");
  });

  it("works for standard UUID", () => {
    const uuid = "550e8400-e29b-41d4-a716-446655440000";
    expect(friendlyOrderId(uuid)).toBe("#ZM-55440000");
  });

  it("returns #ZM- + last 8 chars for short strings", () => {
    expect(friendlyOrderId("abc")).toBe("#ZM-ABC");
    expect(friendlyOrderId("abcdefgh")).toBe("#ZM-ABCDEFGH");
  });
});
