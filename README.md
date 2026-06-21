<div align="center">

# spelunk-md

**Генерирует `CLAUDE.md` для любого проекта — один запуск, без настройки.**

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![OpenRouter](https://img.shields.io/badge/OpenRouter-powered-7C3AED?style=flat-square&logo=openai&logoColor=white)](https://openrouter.ai)
[![CI](https://img.shields.io/github/actions/workflow/status/elev1e1n/spelunk-md/ci.yml?branch=main&style=flat-square)](https://github.com/elev1e1n/spelunk-md/actions)
[![License MIT](https://img.shields.io/github/license/elev1e1n/spelunk-md?style=flat-square)](LICENSE)
[![Release](https://img.shields.io/github/v/release/elev1e1n/spelunk-md?style=flat-square&color=22c55e)](https://github.com/elev1e1n/spelunk-md/releases)

</div>

---

[`CLAUDE.md`](https://docs.anthropic.com/en/docs/claude-code/memory) — файл, который Claude Code читает в начале каждой сессии: архитектура проекта, команды сборки, соглашения по коду, ловушки. Писать его вручную — долго и лениво. `spelunk-md` сканирует проект и генерирует его за секунды через любую модель [OpenRouter](https://openrouter.ai).

## Быстрый старт

```bash
# Установить
go install github.com/elev1e1n/spelunk-md@latest

# Сохранить API-ключ один раз
spelunk-md --api-key sk-or-xxxxxxxxxxxxxxxx

# Запустить в проекте
cd /path/to/your/project
spelunk-md
```

> [!TIP]
> Альтернатива keyring — переменная окружения `OPENROUTER_API_KEY`. Удобно для CI и WSL.

## Как это работает

`spelunk-md` обходит проект и собирает всё, что нужно модели для понимания кодовой базы: файловое дерево с учётом `.gitignore` (включая вложенные), стек технологий по расширениям и манифестам (`go.mod`, `package.json`, `Cargo.toml`), содержимое ключевых конфигов (`Makefile`, `Dockerfile`, `tsconfig.json` и подобных), последние 30 коммитов из git и `README.md` если он есть. Всё это собирается в один структурированный промпт и отправляется в OpenRouter. Результат записывается в `CLAUDE.md`.

Перед перезаписью существующего файла инструмент спросит подтверждение. Флаг `--force` отключает вопрос.

> [!NOTE]
> Модель по умолчанию — `deepseek/deepseek-v4-flash`. Быстрая и дешёвая, хорошо справляется с анализом кода. Для более детального результата попробуй `--model anthropic/claude-sonnet-4.6` или `--model google/gemini-2.5-pro`.

## Примеры

```bash
# Внешний проект
spelunk-md --path ~/projects/my-app

# Другая модель
spelunk-md --model anthropic/claude-opus-4

# Посмотреть промпт без вызова API
spelunk-md --dry-run

# Записать в другой файл
spelunk-md --output docs/PROJECT.md --force

# Удалить сохранённый ключ
spelunk-md --api-key clear
```

Полный список флагов: `spelunk-md --help`.

## Установка

Через `go install`:

```bash
# Последняя версия
go install github.com/elev1e1n/spelunk-md@latest

# Конкретная версия
go install github.com/elev1e1n/spelunk-md@v1.0.0
```

Или собрать из исходников:

```bash
git clone https://github.com/elev1e1n/spelunk-md
cd spelunk-md

# Бинарник соберётся с версией из ближайшего git-тега
just build

# Или напрямую через go build
VERSION=$(git describe --tags --always --dirty)
go build -ldflags "-X main.version=$VERSION" -o spelunk-md .
```

> [!IMPORTANT]
> API-ключ хранится в системном keyring — macOS Keychain, Windows Credential Manager или Linux Secret Service. В коде и файлах проекта ключ не появляется.

## Лицензия

[MIT](LICENSE)
