# spelunk-md

Сканирует кодовую базу и генерирует `CLAUDE.md` через AI (OpenRouter).

## Установка

```bash
go install github.com/elev1e1n/spelunk-md@latest
```

Или собрать вручную:
```bash
git clone https://github.com/elev1e1n/spelunk-md
cd spelunk-md
go build -o spelunk-md .
```

## Использование

**Первый запуск — сохранить API-ключ:**
```bash
spelunk-md --api-key sk-or-xxxxxxxxxxxxxxxx
```
Ключ хранится в системном keyring (macOS Keychain / Windows Credential Manager / Linux Secret Service). В коде и файлах нигде не лежит.

**Генерация CLAUDE.md в текущем проекте:**
```bash
spelunk-md
```

**Указать путь вручную:**
```bash
spelunk-md --path /path/to/project
```

**Другая модель:**
```bash
spelunk-md --model anthropic/claude-opus-4
```

**Посмотреть промпт без вызова API:**
```bash
spelunk-md --dry-run
```

**Удалить сохранённый ключ:**
```bash
spelunk-md --api-key clear
```

## Флаги

| Флаг | По умолчанию | Описание |
|------|-------------|----------|
| `--api-key` | — | OpenRouter API ключ (или `clear` для удаления) |
| `--model` | `deepseek/deepseek-v4-flash` | Модель OpenRouter |
| `--path` | `.` | Путь к проекту |
| `--output` | `CLAUDE.md` | Куда писать результат |
| `--dry-run` | `false` | Показать промпт без вызова API |

## Что анализируется

- Дерево файлов (с учётом `.gitignore`)
- Стек технологий (по расширениям, `go.mod`, `package.json`, `Cargo.toml`, ...)
- Ключевые конфиг-файлы (`Makefile`, `Dockerfile`, `tsconfig.json`, ...)
- Git-история (последние 30 коммитов)
- `README.md` если есть

## Модель по умолчанию

`deepseek/deepseek-v4-flash` — быстрая, дешёвая, хорошо справляется с анализом кода.
Любую другую модель OpenRouter можно передать через `--model`.
