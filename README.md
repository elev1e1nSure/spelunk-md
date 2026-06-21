<div align="center">

# spelunk-md

**Генерирует `CLAUDE.md` для любого проекта — один запуск, без настройки.**

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![OpenRouter](https://img.shields.io/badge/OpenRouter-powered-7C3AED?style=flat-square&logo=openai&logoColor=white)](https://openrouter.ai)
[![License MIT](https://img.shields.io/github/license/elev1e1n/spelunk-md?style=flat-square)](LICENSE)
[![Release](https://img.shields.io/github/v/release/elev1e1n/spelunk-md?style=flat-square&color=22c55e)](https://github.com/elev1e1n/spelunk-md/releases)

</div>

---

[`CLAUDE.md`](https://docs.anthropic.com/en/docs/claude-code/memory) — файл, который Claude Code читает в начале каждой сессии: архитектура, команды, соглашения, ловушки. Писать вручную — долго. `spelunk-md` анализирует проект и генерирует его за секунды через любую модель [OpenRouter](https://openrouter.ai).

## Быстрый старт

```bash
# 1. Установить
go install github.com/elev1e1n/spelunk-md@latest

# 2. Сохранить API-ключ (один раз)
spelunk-md --api-key sk-or-xxxxxxxxxxxxxxxx

# 3. Запустить в проекте
cd /path/to/your/project
spelunk-md
```

> [!TIP]
> Если не хочешь хранить ключ в keyring — просто установи переменную окружения:
> ```bash
> export OPENROUTER_API_KEY=sk-or-xxxxxxxxxxxxxxxx
> ```
> `spelunk-md` подхватит её автоматически.

## Что анализируется

| Источник | Детали |
|----------|--------|
| Файловое дерево | Все файлы с учётом `.gitignore` (включая вложенные) |
| Стек технологий | Языки по доминированию, фреймворки из `go.mod` / `package.json` / `Cargo.toml` |
| Конфиг-файлы | `Makefile`, `justfile`, `Dockerfile`, `tsconfig.json` и др. |
| Git-история | Последние 30 коммитов, ветка, remote URL |
| README | Если есть — извлекается контекст о проекте |

## Флаги

| Флаг | По умолчанию | Описание |
|------|-------------|----------|
| `--api-key` | — | OpenRouter API ключ (или `clear` для удаления) |
| `--model` | `deepseek/deepseek-v4-flash` | Любая модель OpenRouter |
| `--path` | `.` | Путь к проекту |
| `--output` | `CLAUDE.md` | Выходной файл |
| `--timeout` | `120` | Таймаут запроса к API (секунды) |
| `--force` | `false` | Перезаписать без подтверждения |
| `--dry-run` | `false` | Показать промпт без вызова API |

## Примеры

```bash
# Другая модель
spelunk-md --model anthropic/claude-opus-4

# Внешний проект
spelunk-md --path ~/projects/my-app

# Посмотреть промпт перед отправкой
spelunk-md --dry-run

# Записать в другой файл
spelunk-md --output docs/PROJECT.md

# Удалить сохранённый ключ
spelunk-md --api-key clear
```

> [!NOTE]
> Модель по умолчанию — `deepseek/deepseek-v4-flash`. Быстрая, дешёвая, хорошо читает код.
> Для максимального качества можно попробовать `anthropic/claude-sonnet-4.6` или `openai/gpt-5.5`.

## Установка

**Через `go install` (рекомендуется):**
```bash
go install github.com/elev1e1n/spelunk-md@latest
```

**Собрать из исходников:**
```bash
git clone https://github.com/elev1e1n/spelunk-md
cd spelunk-md
go build -o spelunk-md .
```

> [!IMPORTANT]
> API-ключ хранится в системном keyring — macOS Keychain, Windows Credential Manager, Linux Secret Service.
> В коде и файлах проекта ключ не сохраняется.

## Лицензия

[MIT](LICENSE)
<!--  -->