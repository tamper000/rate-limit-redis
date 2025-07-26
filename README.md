# HTTP Rate Limiter Middleware

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

HTTP rate limiting middleware для Go приложений с использованием Redis. Поддерживает Cloudflare и другие proxy-серверы.

## Особенности

- 🚀 Ограничение количества HTTP запросов по IP адресу
- ☁️ Поддержка Cloudflare (`CF-Connecting-IP`) и стандартных proxy (`X-Forwarded-For`)
- 🔧 Два варианта middleware: базовый и с логгированием через `slog`
- 📦 Простая интеграция с любым HTTP роутером
- ⚡ Высокая производительность с использованием Redis pipeline

## Установка

```bash
go get github.com/tamper000/rate-limit-redis@latest
```

## Быстрый старт

```go
package main

import (
    "fmt"
    "net/http"
    "time"

    "github.com/redis/go-redis/v9"
    ratelimit "github.com/tamper000/rate-limit-redis"
)

func main() {
    // Создание Redis клиента
    rdb := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
        DB:   0,
    })

    // Конфигурация rate limiter
    config := ratelimit.Config{
        RedisClient: rdb,
        MaxRequests: 100,        // максимум 100 запросов
        Duration:    time.Hour,  // в час
    }

    // Создание limiter
    limiter := ratelimit.NewLimiter(config)

    // Создание HTTP handler
    mux := http.NewServeMux()
    mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Hello API!")
    })

    // Оборачивание в middleware
    handler := limiter.Middleware(mux)

    fmt.Println("Server starting on :8080")
    http.ListenAndServe(":8080", handler)
}
```

## Использование с логгированием

```go
// Middleware с логгированием через slog
handler := limiter.MiddlewareWithSlog(mux)
```

## Конфигурация

```go
type Config struct {
    RedisClient *redis.Client  // Redis клиент
    MaxRequests int           // Максимальное количество запросов
    Duration    time.Duration // Период для подсчета запросов
}
```

## Поддерживаемые заголовки

Middleware автоматически определяет реальный IP клиента из следующих заголовков:

1. `CF-Connecting-IP` - Cloudflare
2. `X-Forwarded-For` - стандартный заголовок proxy

## Обработка ошибок

- При ошибках Redis middleware продолжает работу (fail-open)
- При превышении лимита возвращается HTTP 429 (Too Many Requests)
- При невозможности определить IP адрес запрос пропускается

## Требования

- Go 1.21+
- Redis сервер
- `github.com/redis/go-redis/v9`

## Лицензия

MIT License - смотрите файл [LICENSE](LICENSE) для подробностей.
