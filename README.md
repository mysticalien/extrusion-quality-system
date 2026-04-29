## Назначение системы

Система предназначена для мониторинга и аналитической обработки параметров экструзионного процесса производства сухих кормов.

## Основные контролируемые параметры

pressure
moisture
barrel_temperature_zone_1
barrel_temperature_zone_2
barrel_temperature_zone_3
screw_speed
drive_load
outlet_temperature

## Роли

operator
technologist
admin

## Пока не делаем

Kafka
Redis
Kubernetes
ML
микросервисы

## Локальный запуск PostgreSQL и backend

Для локального запуска используется файл `.env`. Он не должен добавляться в репозиторий.  
В репозитории хранится только `.env.example`.

Создать локальный `.env`:

```bash
cp .env.example .env
```

После этого нужно заменить значения-заглушки, например POSTGRES_PASSWORD и пароль внутри DATABASE_URL.

Запуск PostgreSQL

```bash
docker compose up -d postgres
```

Проверить, что контейнер запущен:

```bash
docker ps
```

Применение миграции

```bash
docker exec -i extrusion_postgres sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"' < migrations/001_init.sql
```

Проверка таблиц

```bash
docker exec -it extrusion_postgres sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
```

Внутри psql:

```bash
\dt
```

Ожидаемые таблицы:

telemetry_readings
alert_events
quality_index_values
Запуск backend

Backend читает конфигурацию через cleanenv.

Для запуска с .env:

```bash
CONFIG_PATH=.env go run ./cmd/server
```

Проверить health-check:

```bash
curl -i http://localhost:8080/health
```

Проверка записи телеметрии

```bash
curl -i -X POST http://localhost:8080/api/telemetry \
-H "Content-Type: application/json" \
-d '{
"parameterType": "pressure",
"value": 82.5,
"unit": "bar",
"sourceId": "simulator",
"measuredAt": "2026-04-27T18:00:00Z"
}'
```

Проверка данных в PostgreSQL

```bash
docker exec -it extrusion_postgres sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
```

Внутри psql:

```bash
SELECT * FROM telemetry_readings ORDER BY id DESC LIMIT 5;
SELECT * FROM alert_events ORDER BY id DESC LIMIT 5;
SELECT * FROM quality_index_values ORDER BY id DESC LIMIT 5;
```

Возможная проблема: role does not exist

Если при запуске backend появляется ошибка:

FATAL: role "..." does not exist

это означает, что PostgreSQL уже был инициализирован ранее с другим пользователем или другой базой данных. Переменные POSTGRES_USER, POSTGRES_DB и POSTGRES_PASSWORD применяются только при первом создании volume.