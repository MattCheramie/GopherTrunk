import { useEffect, useState } from "react";
import { api } from "../api/client";
import { useStore } from "../store/shared";

// YamlPreview renders the draft as YAML using the server's marshaler, so
// the operator sees exactly the file that will be written. It refreshes
// (debounced) whenever the draft changes.
export function YamlPreview() {
  const config = useStore((s) => s.config);
  const [yaml, setYaml] = useState("");
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    if (!config) return;
    const t = setTimeout(async () => {
      try {
        const r = await api.marshal(config);
        setYaml(r.yaml);
        setErr(null);
      } catch (e) {
        setErr((e as Error).message);
      }
    }, 250);
    return () => clearTimeout(t);
  }, [config]);

  return (
    <div className="flex h-full flex-col">
      <div className="label mb-2">config.yaml preview</div>
      {err ? <p className="text-xs text-err">{err}</p> : null}
      <pre className="flex-1 overflow-auto rounded-md border border-line bg-bg p-3 text-xs font-mono whitespace-pre">
        {yaml}
      </pre>
    </div>
  );
}
