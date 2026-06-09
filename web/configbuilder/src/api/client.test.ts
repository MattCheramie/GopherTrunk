import { afterEach, describe, expect, it, vi } from "vitest";
import { api } from "./client";

afterEach(() => vi.restoreAllMocks());

function mockFetch(bodyObj: unknown) {
  const fn = vi.fn(
    async (_input: RequestInfo | URL, _init?: RequestInit): Promise<Response> =>
      new Response(JSON.stringify(bodyObj), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
  );
  vi.stubGlobal("fetch", fn);
  return fn;
}

describe("RR client credential headers", () => {
  it("rrVerify POSTs with the X-RR-* creds the user is editing", async () => {
    const fetchMock = mockFetch({ ok: true, premium: true, username: "alice" });
    const r = await api.rrVerify({ key: "K", user: "alice", pass: "pw" });
    expect(r.ok).toBe(true);
    expect(r.premium).toBe(true);

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("/api/v1/config/rr/verify");
    expect(init?.method).toBe("POST");
    const h = init?.headers as Record<string, string>;
    expect(h["X-RR-Key"]).toBe("K");
    expect(h["X-RR-User"]).toBe("alice");
    expect(h["X-RR-Pass"]).toBe("pw");
  });

  it("omits empty/blank cred headers", async () => {
    const fetchMock = mockFetch({ results: [] });
    await api.rrStates({ key: "", user: "  ", pass: undefined });
    const h = fetchMock.mock.calls[0][1]?.headers as Record<string, string>;
    expect(h["X-RR-Key"]).toBeUndefined();
    expect(h["X-RR-User"]).toBeUndefined();
    expect(h["X-RR-Pass"]).toBeUndefined();
  });
});
