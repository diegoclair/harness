#!/usr/bin/env python3
"""Weekly review health of one workspace, read from its Claude Code transcripts.

usage: review-rounds.py [WORKSPACE_DIR] [--weeks N]
"""
import argparse
import collections
import datetime
import glob
import json
import os
import re

REVIEWERS = ("architecture-reviewer", "blind-spot-reviewer", "unbiased-reviewer")


def transcripts_dir(workspace):
    slug = re.sub(r"[^A-Za-z0-9]", "-", os.path.abspath(workspace))
    return os.path.join(os.path.expanduser("~/.claude/projects"), slug)


def read_run(path):
    prompt, verdict, started = "", None, None
    for line in open(path, errors="ignore"):
        try:
            entry = json.loads(line)
        except ValueError:
            continue
        started = started or entry.get("timestamp")
        content = entry.get("message", {}).get("content")
        if entry.get("type") == "user" and not prompt:
            prompt = content if isinstance(content, str) else " ".join(
                b.get("text", "") for b in content or [] if isinstance(b, dict))
        if entry.get("type") != "assistant" or not isinstance(content, list):
            continue
        for block in content:
            if not isinstance(block, dict):
                continue
            if block.get("type") == "text":
                text = block["text"]
            elif block.get("name") in ("SendMessage", "SubagentHandback"):
                text = "\n".join(v for v in block.get("input", {}).values() if isinstance(v, str))
            else:
                continue
            found = re.search(r"VERDICT:\s*\**\s*(APPROVE|REJECT|INCOMPLETE)\b(?!\s*\|)", text)
            if found:
                verdict = found.group(1)
            elif re.search(r"^B\d .*:\s*\**(APPROVE|REJECT)", text, re.M):
                verdict = "REJECT" if re.search(r"^B\d .*:\s*\**REJECT", text, re.M) else "APPROVE"
    return prompt, verdict, started


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("workspace", nargs="?", default=os.getcwd())
    parser.add_argument("--weeks", type=int, default=8)
    args = parser.parse_args()
    base = transcripts_dir(args.workspace)
    since = datetime.datetime.now(datetime.timezone.utc) - datetime.timedelta(weeks=args.weeks)

    table = collections.defaultdict(collections.Counter)
    for meta_path in glob.glob(os.path.join(base, "*", "subagents", "*.meta.json")):
        try:
            agent = json.load(open(meta_path)).get("agentType", "")
        except ValueError:
            continue
        if agent not in REVIEWERS:
            continue
        prompt, verdict, started = read_run(meta_path.replace(".meta.json", ".jsonl"))
        if not started or not verdict:
            continue
        when = datetime.datetime.fromisoformat(started.replace("Z", "+00:00"))
        if when < since:
            continue
        week = when.strftime("%G-W%V")
        mode = "re" if re.search(r"RE-REVIEW|re-review|rodada", prompt, re.I) else ("design" if "Mode: DESIGN" in prompt else "first")
        table[(week, agent)][f"{mode}:{verdict}"] += 1

    print(f"{'week':9} {'reviewer':22} {'design ok':>9} {'first ok':>9} {'re-reviews':>10} {'re ok':>6}")
    for (week, agent), c in sorted(table.items()):
        def share(mode):
            total = sum(v for k, v in c.items() if k.startswith(mode + ":"))
            return f"{c[mode + ':APPROVE']}/{total}" if total else "-"
        res = sum(v for k, v in c.items() if k.startswith("re:"))
        print(f"{week:9} {agent:22} {share('design'):>9} {share('first'):>9} {res:>10} {share('re'):>6}")


if __name__ == "__main__":
    main()
