# Reset Generator

Утилита для генерации методов `Reset()` для структур, помеченных комментарием `// generate:reset`.

## Использование

```bash
# Генерация методов Reset() для всех структур в проекте
go run cmd/reset/main.go .

# Генерация для конкретной директории
go run cmd/reset/main.go ./internal/handler
```

## Как это работает

1. Утилита сканирует все `.go` файлы в указанной директории и поддиректориях
2. Ищет структуры с комментарием `// generate:reset` перед объявлением типа
3. Для каждой найденной структуры генерирует метод `Reset()`
4. Сохраняет все сгенерированные методы в файл `reset.gen.go` в том же пакете

## Правила генерации

- **Примитивы** (int, string, bool и т.д.) → нулевые значения (0, "", false)
- **Слайсы** → обрезаются до длины 0 (`slice[:0]`), но не зануляются
- **Мапы** → очищаются через `clear(map)`
- **Указатели** → проверяется на nil, если не nil - сбрасывается значение
- **Вложенные структуры** → если у структуры есть метод `Reset()`, вызывается он, иначе создается новая структура

## Пример

Исходный код:

```go
// generate:reset
type ResetableStruct struct {
    I     int
    Str   string
    StrP  *string
    S     []int
    M     map[string]string
    Child *ResetableStruct
}
```

Сгенерированный метод:

```go
func (r *ResetableStruct) Reset() {
    if r == nil {
        return
    }

    r.I = 0
    r.Str = ""
    if r.StrP != nil {
        *r.StrP = ""
    }
    r.S = r.S[:0]
    clear(r.M)
    if r.Child != nil {
        if resetter, ok := interface{}(r.Child).(interface{ Reset() }); ok {
            resetter.Reset()
        } else {
            *r.Child = ResetableStruct{}
        }
    }
}
```

## Ограничения

- Утилита пропускает директории, начинающиеся с `.` (кроме корневой)
- Пропускает директорию `vendor`
- Не обрабатывает файлы с суффиксом `.gen.go` (сгенерированные файлы)
