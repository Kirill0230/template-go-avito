# Trip Service

Сервис создания, получения и завершения поездок. В лабораторной работе сервис разделён на слои, покрыт тестами и подключён к OpenTelemetry.

## Архитектура

```text
HTTP → usecase → domain
          ↑
     repository
```

- `cmd/trip-service` загружает конфигурацию и запускает приложение;
- `internal/container` создаёт и связывает зависимости;
- `internal/http` содержит handlers, middleware и преобразование ошибок в HTTP;
- `internal/usecase` реализует сценарии работы с поездками;
- `internal/domain` содержит бизнес-правила и переходы статусов;
- `internal/repository` реализует хранение данных в PostgreSQL;
- `internal/infra` инициализирует PostgreSQL, транзакции и OpenTelemetry.
- `internal/worker/positions` опрашивает устаревшие поездки и отправляет команды через `internal/gateway/push`.

Интерфейсы объявлены на стороне потребителя в `contract.go`. Домен не зависит от HTTP и PostgreSQL.

## Запуск

```bash
tripgoctl cluster start
tripgoctl environment start
tripgoctl connect
make migrate
make run
```

Переменные окружения описаны в `.env.example`. Актуальные адреса окружения выводит `tripgoctl connect`.

## Проверка

```bash
make lint
make test
make test-integration
```

Unit-тесты запускаются с race detector. Интеграционные тесты работают с PostgreSQL и проверяют сохранение данных и откат транзакции.

## Опрос устаревших поездок

```text
PostgreSQL → сканер → ограниченная очередь → worker pool → Push Service
```

Сканер раз в `POSITION_CHECK_INTERVAL` выбирает активные поездки, которые не присылали координаты дольше `POSITION_STALE_AFTER`. Выборка читается keyset-батчами по `POSITION_SCAN_BATCH_SIZE`, без `OFFSET`. Пул из `POSITION_WORKERS` обработчиков отправляет водителям `REQUEST_POSITION`. Размер очереди задаёт `POSITION_QUEUE_SIZE`, таймаут одного вызова — `PUSH_REQUEST_TIMEOUT`, адрес сервиса — `PUSH_HTTP_URL`.

Одна поездка не ставится в очередь повторно. При получении координаты ожидающая задача отменяется. Очередь приоритетная: первой обрабатывается поездка, которая молчит дольше всех. При заполнении более старая задача вытесняет самую свежую, иначе новая задача получает результат `dropped`. После первого `dropped` сканер прекращает текущий обход и начинает заново на следующем тике.

При остановке сначала закрывается HTTP-сервер, затем сканер, после чего воркеры дорабатывают очередь в пределах общего `SHUTDOWN_TIMEOUT`; только после этого закрываются Push-клиент и PostgreSQL.

## Observability

Сервис отправляет телеметрию по OTLP:

- JSON-логи `slog` содержат `trace_id` и `span_id`;
- HTTP и PostgreSQL запросы представлены отдельными spans;
- собираются HTTP и DB метрики;
- бизнес-метрики считают созданные и завершённые поездки.

### Логи

![Структурные JSON-логи](docs/images/structured-logs.png)

### Трейс в Jaeger

![Трейс завершения поездки](docs/images/jaeger-trace.png)

### Метрики в Grafana

Созданные поездки:

![Метрика созданных поездок](docs/images/metrics-trips-created.png)

Завершённые поездки:

![Метрика завершённых поездок](docs/images/metrics-trips-completed.png)

HTTP-запросы:

![HTTP-метрики](docs/images/metrics-http.png)

Операции PostgreSQL:

![Метрики PostgreSQL](docs/images/metrics-database.png)

## Мутационное тестирование

Пакеты `internal/domain` и `internal/usecase` проверены с помощью Gremlins.

| Запуск | Killed | Lived | Not covered | Покрытие мутаций |
|---|---:|---:|---:|---:|
| До дополнительного теста | 17 | 0 | 5 | 77,27% |
| После дополнительного теста | 22 | 0 | 0 | 100% |

После добавления теста повторного идемпотентного запроса все найденные мутации уничтожаются тестами. Отчёты сохранены в `docs/mutation`.
