import { describe, it, expect, beforeEach } from "vitest";
import { useShared, selectClientConfig } from "./shared";

// Regression coverage for issue #290: selectClientConfig must return a
// referentially-stable object so callers can put `cfg` in useEffect
// dependency arrays without provoking a render loop.
describe("selectClientConfig", () => {
  beforeEach(() => {
    useShared.setState({
      serverURL: "http://localhost:8080",
      token: null,
      wsStatus: "idle",
    });
  });

  it("returns the same reference on repeated calls", () => {
    const a = selectClientConfig(useShared.getState());
    const b = selectClientConfig(useShared.getState());
    expect(b).toBe(a);
  });

  it("keeps the same reference after an unrelated store change", () => {
    const before = selectClientConfig(useShared.getState());
    useShared.setState({ wsStatus: "connecting" });
    const after = selectClientConfig(useShared.getState());
    expect(after).toBe(before);
  });

  it("returns a new reference when serverURL changes", () => {
    const before = selectClientConfig(useShared.getState());
    useShared.setState({ serverURL: "http://other:9090" });
    const after = selectClientConfig(useShared.getState());
    expect(after).not.toBe(before);
    expect(after.baseURL).toBe("http://other:9090");
  });

  it("returns a new reference when the token changes", () => {
    const before = selectClientConfig(useShared.getState());
    useShared.setState({ token: "secret" });
    const after = selectClientConfig(useShared.getState());
    expect(after).not.toBe(before);
    expect(after.token).toBe("secret");
  });

  it("maps a null serverURL to an empty baseURL", () => {
    useShared.setState({ serverURL: null });
    expect(selectClientConfig(useShared.getState()).baseURL).toBe("");
  });
});
