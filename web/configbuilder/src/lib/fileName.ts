// downloadName derives a .yaml filename for the browser download from the
// current file path (basename), defaulting to config.yaml for an unsaved
// draft and appending .yaml when the basename lacks a yaml extension.
export function downloadName(path: string): string {
  const base = path ? (path.split(/[/\\]/).pop() ?? "") : "";
  if (!base) return "config.yaml";
  return /\.ya?ml$/i.test(base) ? base : base + ".yaml";
}

// joinPath joins a directory and filename with a forward slash, trimming a
// trailing separator on the dir.
export function joinPath(dir: string, name: string): string {
  if (!dir) return name;
  return dir.replace(/[/\\]+$/, "") + "/" + name;
}

// pathCollides reports whether target matches an already-discovered file —
// used to warn that saving over it re-marshals and strips YAML comments.
export function pathCollides(files: { path: string }[], target: string): boolean {
  return files.some((f) => f.path === target);
}
