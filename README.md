## Запуск

```bash
tripgoctl cluster start
tripgoctl environment start
tripgoctl connect
make migrate
make run
```

## Переменные окружения

| Переменная | Пример |
|---|---|
| `HTTP_ADDR` | `:8080` |
| `LOG_LEVEL` | `info` |
| `SHUTDOWN_TIMEOUT` | `10s` |
| `DATABASE_URL` | `postgres://user:password@localhost:5432/db?sslmode=disable` |
| `DATABASE_MAX_CONNS` | `10` |
| `DATABASE_MIN_CONNS` | `2` |
| `DATABASE_MAX_CONN_LIFETIME` | `30m` |
| `DATABASE_CONNECT_TIMEOUT` | `5s` |
| `DATABASE_QUERY_TIMEOUT` | `3s` |

Все обязательны, пример — `.env.example`.


## Решения

**Уровень изоляции — READ COMMITTED.**

**Менеджер транзакций.** `Do` открывает транзакцию и кладёт её в `context`; репозитории берут её оттуда, а без неё работают через пул.

**Одна активная поездка на водителя** — частичный уникальный индекс `ON trips (driver_id) WHERE status = 'active'`.

**Идемпотентность** — таблица `idempotency_keys` (ключ → поездка), ключ занимается через `INSERT ... ON CONFLICT`. Ключ живёт **24 часа**.

## Docker

```bash
docker build -t trip-service .
Размер образа: **17.9 МБ**.
```

