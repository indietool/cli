#!/usr/bin/env python3
"""Emit a KEY=VALUE dotenv blob from `indietool secret export @db --json` on stdin.

This is the emitter half of the Hermes `secrets.command` source recipe:

    indietool secret export @hermes --json | toenv.py

One line per secret (NAME=value). Multiline values are skipped loudly on
stderr: the dotenv map format is one line per variable, so multiline secret
material must be stored base64-encoded and decoded by the consumer.
"""
import json
import sys


def main() -> int:
    try:
        data = json.load(sys.stdin)
    except json.JSONDecodeError as exc:
        print(f"toenv: invalid JSON on stdin: {exc}", file=sys.stderr)
        return 1
    for _db, items in data.get("databases", {}).items():
        for item in items:
            value = item.get("value", "")
            if "\n" in value or "\r" in value:
                print(f"toenv: skip {item.get('name', '?')}: multiline value", file=sys.stderr)
                continue
            sys.stdout.write(f"{item['name']}={value}\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
