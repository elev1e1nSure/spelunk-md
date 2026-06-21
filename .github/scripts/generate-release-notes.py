#!/usr/bin/env python3
"""Generate Russian release notes via OpenRouter for the current tag."""

import json
import os
import subprocess
import urllib.request


def main():
    api_key = os.environ.get("OPENROUTER_API_KEY")
    if not api_key:
        raise SystemExit("OPENROUTER_API_KEY is not set")

    model = os.environ.get("MODEL", "deepseek/deepseek-v4-flash")

    # Collect commits since the previous tag, or all commits for the first release.
    try:
        prev_tag = subprocess.check_output(
            ["git", "describe", "--tags", "--abbrev=0", "HEAD~1"],
            text=True,
            stderr=subprocess.DEVNULL,
        ).strip()
        commits = subprocess.check_output(
            ["git", "log", "--pretty=format:%s", f"{prev_tag}..HEAD"],
            text=True,
        )
    except subprocess.CalledProcessError:
        commits = subprocess.check_output(
            ["git", "log", "--pretty=format:%s"],
            text=True,
        )

    prompt = f"""Ты пишешь release notes для CLI-утилиты spelunk-md на русском языке.
Не используй эмодзи.
Не пиши технических деталей вроде названий коммитов, имён файлов или внутренних механизмов.
Опиши только то, что полезно пользователю: новые возможности, улучшения, исправления.
Используй заголовки "Что нового", "Улучшения", "Исправления" где уместно.
Будь кратким и по делу.

Список изменений:
{commits}
"""

    payload = {
        "model": model,
        "messages": [{"role": "user", "content": prompt}],
    }

    req = urllib.request.Request(
        "https://openrouter.ai/api/v1/chat/completions",
        data=json.dumps(payload).encode(),
        headers={
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
        },
        method="POST",
    )

    with urllib.request.urlopen(req, timeout=120) as resp:
        data = json.loads(resp.read().decode())
        content = data["choices"][0]["message"]["content"]

    with open("release-notes.md", "w", encoding="utf-8") as f:
        f.write(content)

    # Expose the notes to subsequent workflow steps.
    github_env = os.environ.get("GITHUB_ENV")
    if github_env:
        with open(github_env, "a", encoding="utf-8") as f:
            f.write("RELEASE_NOTES<<EOF\n")
            f.write(content)
            f.write("\nEOF\n")


if __name__ == "__main__":
    main()
