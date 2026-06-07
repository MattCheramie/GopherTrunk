import type { DMRBandPlan } from "../api/types";

// DMR Tier III band plans must set exactly one of Linear / Table (enforced
// by config.Validate; see internal/config/config.go validateTrunking). These
// pure helpers map between that shape and a 3-way UI mode so the editor can
// switch cleanly without leaving both branches populated.
export type DMRBandPlanMode = "none" | "linear" | "table";

export function dmrBandPlanMode(bp: DMRBandPlan | null | undefined): DMRBandPlanMode {
  if (!bp) return "none";
  if (bp.Linear) return "linear";
  if (bp.Table) return "table";
  return "none";
}

// dmrBandPlanForMode returns the band plan for the chosen mode, preserving
// any previously-entered values for that branch so toggling modes is
// non-destructive within a session.
export function dmrBandPlanForMode(
  mode: DMRBandPlanMode,
  prev: DMRBandPlan | null | undefined,
): DMRBandPlan | null {
  switch (mode) {
    case "none":
      return null;
    case "linear":
      return { Linear: prev?.Linear ?? { BaseHz: 0, SpacingHz: 0, Offset: 1 }, Table: null };
    case "table":
      return { Linear: null, Table: prev?.Table ?? [] };
  }
}
