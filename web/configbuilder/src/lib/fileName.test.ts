import { describe, expect, it } from "vitest";
import { downloadName, joinPath, pathCollides } from "./fileName";

describe("downloadName", () => {
  it("defaults to config.yaml for an empty path", () => {
    expect(downloadName("")).toBe("config.yaml");
  });
  it("keeps a yaml basename", () => {
    expect(downloadName("/etc/gophertrunk/prod.yaml")).toBe("prod.yaml");
    expect(downloadName("C:\\cfg\\thing.yml")).toBe("thing.yml");
  });
  it("appends .yaml when missing", () => {
    expect(downloadName("/cfg/weird")).toBe("weird.yaml");
  });
});

describe("joinPath", () => {
  it("joins with a single slash", () => {
    expect(joinPath("/a/b", "c.yaml")).toBe("/a/b/c.yaml");
    expect(joinPath("/a/b/", "c.yaml")).toBe("/a/b/c.yaml");
    expect(joinPath("", "c.yaml")).toBe("c.yaml");
  });
});

describe("pathCollides", () => {
  const files = [{ path: "/a/config.yaml" }, { path: "/a/other.yaml" }];
  it("detects an existing path", () => {
    expect(pathCollides(files, "/a/config.yaml")).toBe(true);
    expect(pathCollides(files, "/a/new.yaml")).toBe(false);
  });
});
