import type { NXDNBandPlan } from "../api/types";

// NXDN band plans, like DMR's, must set exactly one of Linear / Table
// (config.Validate). These helpers map that shape to a 3-way UI mode so the
// editor switches cleanly without leaving both branches populated.
export type NXDNBandPlanMode = "none" | "linear" | "table";

export function nxdnBandPlanMode(bp: NXDNBandPlan | null | undefined): NXDNBandPlanMode {
  if (!bp) return "none";
  if (bp.Linear) return "linear";
  if (bp.Table) return "table";
  return "none";
}

// nxdnBandPlanForMode returns the band plan for the chosen mode, keeping the
// values already entered for that branch so toggling modes is
// non-destructive within a session. NXDN sites usually number channels from
// 1, so a new linear plan starts at offset 1.
export function nxdnBandPlanForMode(
  mode: NXDNBandPlanMode,
  prev: NXDNBandPlan | null | undefined,
): NXDNBandPlan | null {
  switch (mode) {
    case "none":
      return null;
    case "linear":
      return { Linear: prev?.Linear ?? { BaseHz: 0, SpacingHz: 0, Offset: 1 }, Table: null };
    case "table":
      return { Linear: null, Table: prev?.Table ?? [] };
  }
}
