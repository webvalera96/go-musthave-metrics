# Staticlint - singlechecker для статического анализа

Singlechecker запускает один анализатор для проверки кода проекта go-musthave-metrics.

## Установка

```bash
go build -o staticlint cmd/staticlint/main.go
```

## Использование

```bash
# Анализ всего проекта
./staticlint ./...

# Анализ конкретной директории
./staticlint ./cmd/agent/...

# Анализ конкретного пакета
./staticlint ./internal/handler/...
```

Или напрямую через go run:

```bash
go run cmd/staticlint/main.go ./...
```

## Включенные анализаторы

### 1. Собственный анализатор

- **exitcheck**:
  - запрещает `panic(...)` **везде**
  - запрещает `os.Exit(...)` и `log.Fatal(...)` **везде, кроме `main.main`**

## Примеры использования

### Проверка всего проекта

```bash
./staticlint ./...
```
