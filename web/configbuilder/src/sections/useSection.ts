import { useStore } from "../store/shared";
import type { GTConfig } from "../api/types";

// useSection returns the current value of a typed config section plus a
// setter that patches just that section into the draft. The key is the
// capitalized Go field name (the JSON key the daemon emits).
export function useSection<K extends keyof GTConfig>(
  key: K,
): [GTConfig[K], (v: GTConfig[K]) => void] {
  const value = useStore((s) => (s.config ? s.config[key] : null)) as GTConfig[K];
  const patch = useStore((s) => s.patchSection);
  return [value, (v) => patch(key, v)];
}
