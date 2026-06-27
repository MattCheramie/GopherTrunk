import { useStore } from "../store/shared";
import type { Param } from "../api/types";

// ParamField renders one mode input as the right control for its kind,
// matching the house form styles (.label/.input/.help).
export function ParamField({ param }: { param: Param }) {
  const value = useStore((s) => s.params[param.name] ?? "");
  const file = useStore((s) => s.files[param.name]);
  const uploading = useStore((s) => s.uploading[param.name]);
  const setParam = useStore((s) => s.setParam);
  const uploadFor = useStore((s) => s.uploadFor);

  if (param.kind === "file") {
    return (
      <label className="block">
        <span className="label">
          {param.label}
          {param.required ? " *" : ""}
        </span>
        <input
          type="file"
          className="input"
          onChange={(e) => {
            const f = e.target.files?.[0];
            if (f) void uploadFor(param.name, f);
          }}
        />
        {uploading ? (
          <p className="help mt-1 text-accent">uploading…</p>
        ) : file ? (
          <p className="help mt-1 text-ok">
            ✓ {file.name} ({file.size.toLocaleString()} bytes)
          </p>
        ) : null}
        {param.help ? <p className="help mt-1">{param.help}</p> : null}
      </label>
    );
  }

  if (param.kind === "bool") {
    return (
      <label className="flex items-start gap-2 py-1">
        <input
          type="checkbox"
          className="mt-1"
          checked={value === "true"}
          onChange={(e) => setParam(param.name, e.target.checked ? "true" : "false")}
        />
        <span>
          <span className="text-sm">{param.label}</span>
          {param.help ? <p className="help">{param.help}</p> : null}
        </span>
      </label>
    );
  }

  if (param.kind === "select") {
    return (
      <label className="block">
        <span className="label">{param.label}</span>
        <select className="input" value={value} onChange={(e) => setParam(param.name, e.target.value)}>
          {(param.options ?? []).map((o) => (
            <option key={o} value={o}>
              {o}
            </option>
          ))}
        </select>
        {param.help ? <p className="help mt-1">{param.help}</p> : null}
      </label>
    );
  }

  // string | int | outfile
  return (
    <label className="block">
      <span className="label">
        {param.label}
        {param.required ? " *" : ""}
      </span>
      <input
        className="input"
        type={param.kind === "int" ? "number" : "text"}
        value={value}
        placeholder={param.default}
        onChange={(e) => setParam(param.name, e.target.value)}
      />
      {param.help ? <p className="help mt-1">{param.help}</p> : null}
    </label>
  );
}
