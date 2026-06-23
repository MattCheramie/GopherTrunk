#!/usr/bin/env python3
"""PreToolUse guard for GitHub issue closes.

Stops Claude from closing a GitHub issue as *completed* ("fixed") without a
human in the loop. This exists because issue #764 was closed twice with a
comment that merely repeated the fix description while the reported symptom
was still live — see issue #771 and CLAUDE.md "Issue-closing policy".

It only gates closes-as-completed. Closing as not_planned or duplicate, and
every non-close issue write (create, comment, label, reopen), pass straight
through. On a close-as-completed it returns an "ask" decision so the close
still happens once a human confirms the fix is genuinely verified.

Reads the PreToolUse hook payload on stdin; writes a hook decision on stdout.
Always exits 0 — a guard must never become a hard failure that blocks work.
"""
import json
import sys


def main() -> int:
    try:
        data = json.load(sys.stdin)
    except Exception:
        return 0  # never block on a malformed/empty payload

    ti = data.get("tool_input", {}) or {}
    is_close_completed = (
        ti.get("method") == "update"
        and ti.get("state") == "closed"
        and ti.get("state_reason") in (None, "", "completed")
    )
    if not is_close_completed:
        return 0  # allow: not a close-as-completed

    num = ti.get("issue_number", "?")
    reason = (
        f"About to close issue #{num} as completed. Confirm the fix is VERIFIED first: "
        "a failing-first regression test now passes AND the reporter has confirmed the fix "
        "(or you reproduced the original symptom and this change resolves it). If it is not "
        "yet verified, cancel this close and post a status comment instead, leaving the issue "
        "open. Do not re-post the original fix description as the close justification — address "
        "the latest follow-up. See CLAUDE.md \"Issue-closing policy\"."
    )
    print(json.dumps({
        "hookSpecificOutput": {
            "hookEventName": "PreToolUse",
            "permissionDecision": "ask",
            "permissionDecisionReason": reason,
        }
    }))
    return 0


if __name__ == "__main__":
    sys.exit(main())
