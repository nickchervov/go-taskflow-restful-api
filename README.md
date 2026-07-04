# TaskFlow — Async Task Processing API

Учебный проект на Go. REST API, которое принимает задачи, ставит их в очередь, обрабатывает фоновыми воркерами и сохраняет результат.

**Стек:** Go 1.25.6, PostgreSQL, Redis, Docker

---

## Чему я здесь учился

- Строить REST API на стандартной библиотеке `net/http` (без фреймворков)
- Работать с PostgreSQL через `pgx` (пул соединений, named args, JSONB)
- Использовать Redis как очередь задач (`LPush` / `BRPop`), кеш и rate limiter
- Писать worker pool на горутинах с graceful shutdown
- Настраивать Docker Compose для связки Go + Postgres + Redis
- Использовать `log/slog` для структурированного логирования

---

## Как запустить

```bash
docker compose up --build
```

Поднимутся три контейнера:
- `db` — PostgreSQL 16 (:5432)
- `redis` — Redis 7 (:6379)
- `app` — само приложение (:8080)

После запуска можно сразу делать запросы.

---

## API

### Создать задачу

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{"type": "send_email", "payload": {"to": "user@example.com", "subject": "Hello"}}'
```

Типы задач: `send_email`, `resize_image`, `generate_report`.

### Получить статус и результат

```bash
curl http://localhost:8080/api/v1/tasks/<uuid>
```

Сначала проверяет кеш в Redis, если нет — идёт в БД.

### Список всех задач

```bash
curl http://localhost:8080/api/v1/tasks
```

### Отменить задачу

```bash
curl -X DELETE http://localhost:8080/api/v1/tasks/<uuid>
```

### Перезапустить

```bash
curl -X POST http://localhost:8080/api/v1/tasks/<uuid>/retry
```

---

## Как это устроено внутри

```
POST /api/v1/tasks
    │
    ├── Сохраняем задачу в PostgreSQL (status: pending)
    ├── Отправляем в Redis очередь (queue:pending)
    └── Возвращаем 201

Воркер (5 горутин)
    │
    ├── BRPOP из очереди
    ├── Статус → running
    ├── Выполняем задачу (по типу)
    ├── Статус → completed / failed
    ├── Кешируем результат в Redis
    └── Пишем в БД
```

Если задача падает — воркер делает повторную попытку (до 3 раз).

---

Проект писал в обучающих целях. Буду рад фидбеку и замечаниям.
