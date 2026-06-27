import { api } from "../api/client";
import { selectClientConfig, useShared } from "../store/shared";
import { useDataPoll } from "./useDataPoll";

// useActiveCallsPoll keeps the shared-store activeCalls snapshot fresh
// while a panel that follows live calls is mounted. The Active and
// Dashboard panels each run their own activeCalls poll, but those only
// tick on their own routes — so a spectrum panel opened directly (e.g.
// #/mixer) was left reading whatever snapshot happened to be in the
// store. A call that had already ended still looked active, freezing the
// follow offset on a single grant frequency ("following call" forever).
//
// Reuses useDataPoll, which swallows fetch errors and only delivers a
// changed snapshot, so this adds no render churn beyond a real update.
export function useActiveCallsPoll(intervalMs = 2_000) {
  const cfg = useShared(selectClientConfig);
  const setActiveCalls = useShared((s) => s.setActiveCalls);
  useDataPoll({
    fetcher: () => api.activeCalls(cfg),
    onData: setActiveCalls,
    intervalMs,
    resetKey: cfg.baseURL,
  });
}
