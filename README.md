# Информационно-аналитическая система контроля качества экструзии кормов

Система предназначена для мониторинга параметров технологического процесса экструзии сухих кормов, выявления отклонений, обнаружения аномалий и расчёта итогового индекса качества процесса.

Проект включает backend на Go, PostgreSQL, Kafka/MQTT-интеграцию, ролевую модель доступа, веб-интерфейс оператора и демонстрационные сценарии для защиты.

---

## Возможности системы

Система позволяет:

- принимать телеметрию технологических параметров;
- хранить последние и исторические значения измерений;
- сравнивать параметры с технологическими уставками;
- создавать события уровней `предупреждение` и `критично`;
- обнаруживать аномалии процесса;
- рассчитывать индекс качества экструзии;
- разграничивать доступ по ролям пользователей;
- управлять уставками и весами параметров;
- просматривать графики истории измерений;
- демонстрировать normal, warning, critical и anomaly-сценарии.

---

## Скриншоты интерфейса

Ниже представлены основные экраны системы: авторизация, мониторинг технологических параметров, события, аномалии, история телеметрии, настройка уставок и административные разделы.

### Авторизация

Окно входа в систему. Доступ к функциям зависит от роли пользователя: оператор, технолог или администратор.

![Окно входа](docs/assets/screens/entry.png)

### Главная панель мониторинга

Основной экран системы с текущим индексом качества и последними значениями технологических параметров экструзионного процесса.

![Главная панель мониторинга](docs/assets/screens/dashboard-overview.png)

### Активные события

Раздел с предупреждениями и критическими отклонениями параметров от заданных уставок. Оператор может подтверждать события после просмотра.

![Активные события](docs/assets/screens/events.png)

### Активные аномалии

Раздел отображает обнаруженные аномалии: резкие скачки, дрейф параметров и комбинированные признаки нестабильности процесса.

![Аномалии](docs/assets/screens/anomaly.png)

### История телеметрии

Экран просмотра исторических измерений по выбранному параметру с графиком и ограничением количества записей.

![История телеметрии](docs/assets/screens/telemetry-history.png)

### Настройка уставок

Раздел для технолога и администратора, где задаются границы нормального, предупредительного и критического состояния параметров.

![Настройка уставок](docs/assets/screens/setpoints.png)

### Управление весами качества

Раздел настройки весов параметров, влияющих на итоговый интегральный индекс качества.

![Управление качеством](docs/assets/screens/quality.png)

### Kafka UI

Интерфейс Kafka UI используется для визуальной проверки топиков, сообщений телеметрии и состояния брокера сообщений.

![Kafka UI](docs/assets/screens/kafka.png)

### Docker Desktop

Контейнеры инфраструктуры системы: PostgreSQL, Kafka, Kafka UI, миграции и сервис приложения.

![Docker Desktop](docs/assets/screens/docker.png)

### DataGrip

ER-диаграмма и структура базы данных в DataGrip. Используется для проверки таблиц, связей и ограничений.

![DataGrip](docs/assets/screens/datagrip.png)

### Пример таблицы уставок

Пример данных таблицы `setpoints`, где хранятся диапазоны допустимых значений технологических параметров.

![Таблица setpoints](docs/assets/screens/table-example.png)
---

## Технологический стек

Backend:

- Go
- net/http
- PostgreSQL
- pgx
- goose migrations
- Kafka
- MQTT
- JWT-аутентификация
- bcrypt-хеширование паролей
- unit-тесты и integration-тесты

Frontend:

- HTML
- CSS
- JavaScript
- Canvas-график истории телеметрии

Инфраструктура:

- Docker
- Docker Compose
- Makefile

---

## Архитектура проекта

Проект разделён по слоям:

```text
cmd/
  server/              точка входа backend-сервера
  simulator/           симулятор телеметрии

internal/
  adapters/
    http/              HTTP-обработчики
    kafka/             Kafka producer/consumer
    mqtt/              MQTT subscriber
    postgres/          PostgreSQL-репозитории

  app/
    server/            сборка зависимостей, запуск HTTP-сервера
    simulator/         запуск симулятора

  config/              загрузка и валидация конфигурации

  domain/              доменные модели и правила

  ports/               интерфейсы репозиториев и сервисов

  security/
    password/          bcrypt-хеширование паролей
    token/             JWT-токены

  usecase/
    auth/              сценарии авторизации
    telemetry/         обработка телеметрии
    anomalies/         обнаружение аномалий
    quality/           расчёт индекса качества
```

Основная цепочка обработки:

```text
телеметрия → проверка уставок → событие/аномалия → индекс качества → отображение в UI
```

---

## Роли пользователей

В системе предусмотрены три роли.

### Оператор

Может:

- просматривать текущие параметры;
- видеть активные события;
- подтверждать события.

Не может:

- изменять уставки;
- управлять пользователями;
- изменять веса качества.

### Технолог

Может:

- просматривать параметры;
- видеть события и аномалии;
- просматривать историю телеметрии;
- изменять технологические уставки;
- изменять веса параметров для индекса качества.

### Администратор

Может:

- управлять пользователями;
- менять роли пользователей;
- активировать и деактивировать пользователей;
- сбрасывать пароли;
- выполнять действия технолога.

---

## Быстрый запуск

### 1. Склонировать проект

```bash
git clone <repository-url>
cd extrusion-quality-system
```

### 2. Подготовить `.env`

Можно взять пример:

```bash
cp .env.example .env
```

Пример важных переменных:

```env
SERVER_ADDR=:8080

DATABASE_URL=postgres://postgres:postgres@postgres:5432/extrusion_quality?sslmode=disable

JWT_SECRET=local-dev-secret-change-me
JWT_TOKEN_TTL=24h
AUTH_TOKEN_ISSUER=extrusion-quality-system
AUTH_BCRYPT_COST=10

KAFKA_ENABLED=true
KAFKA_BROKERS=kafka:9092
KAFKA_TELEMETRY_TOPIC=extrusion.telemetry.raw
KAFKA_CONSUMER_GROUP=extrusion-quality-service

MQTT_ENABLED=true
MQTT_BROKER_URL=tcp://mqtt:1883
MQTT_TELEMETRY_TOPIC=extrusion/telemetry
MQTT_CLIENT_ID=extrusion-quality-server
MQTT_QOS=0
```

### 3. Запустить систему

```bash
make compose-up
```

После запуска открыть:

```text
http://localhost:8080
```

---

## Остановка системы

```bash
make compose-down
```

Если нужно полностью очистить контейнеры и volumes:

```bash
docker compose down -v
```

---

## Миграции базы данных

Миграции запускаются автоматически через compose-сервис `migrate`.

Ручной запуск:

```bash
goose -dir migrations postgres "$DATABASE_URL" up
```

---

## Тестовые пользователи

После применения seed-данных доступны пользователи для демонстрации.

```text
operator      — оператор
technologist  — технолог
admin         — администратор
```

Пароли зависят от seed-миграции проекта. Если пароль был изменён через UI, можно сбросить базу:

```bash
docker compose down -v
make compose-up
```

---

## API

### Авторизация

```http
POST /api/login
```

Пример:

```bash
curl -s http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "technologist",
    "password": "technologist-password"
  }'
```

### Текущий пользователь

```http
GET /api/me
```

```bash
curl -s http://localhost:8080/api/me \
  -H "Authorization: Bearer $TOKEN"
```

### Отправка телеметрии

```http
POST /api/telemetry
```

```bash
NOW=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

curl -s http://localhost:8080/api/telemetry \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"parameterType\": \"pressure\",
    \"value\": 95,
    \"unit\": \"bar\",
    \"sourceId\": \"demo\",
    \"measuredAt\": \"$NOW\"
  }"
```

### Последние значения телеметрии

```http
GET /api/telemetry/latest
```

### История телеметрии

```http
GET /api/telemetry/history?parameter=pressure&limit=30
```

### Активные события

```http
GET /api/events/active
```

### Активные аномалии

```http
GET /api/anomalies/active
```

### Индекс качества

```http
GET /api/quality/latest
```

---

## Тестирование

Запуск всех тестов:

```bash
go test ./...
```

Запуск с покрытием:

```bash
go test ./... -cover
```

Генерация HTML-отчёта покрытия:

```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

Покрытые части:

- доменные правила уставок;
- расчёт индекса качества;
- обнаружение аномалий;
- сценарии обработки телеметрии;
- авторизация;
- JWT-токены;
- bcrypt-хеширование;
- проверка прав доступа;
- минимальная integration-цепочка.

---

## Демонстрационные сценарии

Демонстрационные сценарии нужны, чтобы за 3–5 минут показать работу всей системы.

Перед запуском сценариев нужно получить токен:

```bash
TOKEN=$(curl -s http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"username":"technologist","password":"technologist-password"}' \
  | jq -r '.token')
```

---

## Сценарий 1. Нормальный режим

Показывает:

- параметры в норме;
- индекс качества высокий;
- активных событий нет.

```bash
NOW=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

curl -s http://localhost:8080/api/telemetry \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"parameterType\":\"pressure\",
    \"value\":95,
    \"unit\":\"bar\",
    \"sourceId\":\"demo-normal\",
    \"measuredAt\":\"$NOW\"
  }"

curl -s http://localhost:8080/api/telemetry \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"parameterType\":\"moisture\",
    \"value\":22,
    \"unit\":\"percent\",
    \"sourceId\":\"demo-normal\",
    \"measuredAt\":\"$NOW\"
  }"

curl -s http://localhost:8080/api/telemetry \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"parameterType\":\"drive_load\",
    \"value\":45,
    \"unit\":\"percent\",
    \"sourceId\":\"demo-normal\",
    \"measuredAt\":\"$NOW\"
  }"
```

---

## Сценарий 2. Предупреждение

Показывает:

- давление вышло в warning-диапазон;
- создалось событие;
- индекс качества снизился;
- оператор может подтвердить событие.

```bash
NOW=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

curl -s http://localhost:8080/api/telemetry \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"parameterType\":\"pressure\",
    \"value\":88,
    \"unit\":\"bar\",
    \"sourceId\":\"demo-warning\",
    \"measuredAt\":\"$NOW\"
  }"
```

После этого в UI открыть вкладку **События** и нажать **Подтвердить**.

---

## Сценарий 3. Критическое отклонение

Показывает:

- параметр находится в critical-диапазоне;
- создаётся критическое событие;
- индекс качества снижается сильнее.

```bash
NOW=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

curl -s http://localhost:8080/api/telemetry \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"parameterType\":\"pressure\",
    \"value\":145,
    \"unit\":\"bar\",
    \"sourceId\":\"demo-critical\",
    \"measuredAt\":\"$NOW\"
  }"
```

---

## Сценарий 4. Аномалия процесса

Показывает:

- влажность падает;
- давление растёт;
- нагрузка привода растёт;
- система обнаруживает комбинированный риск процесса.

```bash
for i in 0 1 2 3 4
do
  TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

  PRESSURE=$((90 + i * 3))
  MOISTURE=$(awk "BEGIN {print 23 - $i * 0.6}")
  DRIVE_LOAD=$((45 + i * 3))

  curl -s http://localhost:8080/api/telemetry \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"parameterType\":\"pressure\",\"value\":$PRESSURE,\"unit\":\"bar\",\"sourceId\":\"demo-anomaly\",\"measuredAt\":\"$TS\"}" > /dev/null

  curl -s http://localhost:8080/api/telemetry \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"parameterType\":\"moisture\",\"value\":$MOISTURE,\"unit\":\"percent\",\"sourceId\":\"demo-anomaly\",\"measuredAt\":\"$TS\"}" > /dev/null

  curl -s http://localhost:8080/api/telemetry \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"parameterType\":\"drive_load\",\"value\":$DRIVE_LOAD,\"unit\":\"percent\",\"sourceId\":\"demo-anomaly\",\"measuredAt\":\"$TS\"}" > /dev/null

  sleep 1
done
```

После выполнения открыть вкладку **События** или **Аномалии**.

---

## Сценарий 5. Проверка ролевого доступа

### Оператор

Войти как оператор.

Показать:

- доступен обзор;
- доступны события;
- можно подтвердить событие;
- вкладка уставок недоступна.

### Технолог

Войти как технолог.

Показать:

- доступна история;
- доступны аномалии;
- доступна настройка уставок;
- можно изменить границы параметра.

---

## Удобный demo-скрипт

Для защиты можно добавить файл:

```text
scripts/demo.sh
```

Пример запуска:

```bash
./scripts/demo.sh normal
./scripts/demo.sh warning
./scripts/demo.sh critical
./scripts/demo.sh anomaly
```

Пример содержимого:

```bash
#!/usr/bin/env bash

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
USERNAME="${USERNAME:-technologist}"
PASSWORD="${PASSWORD:-technologist-password}"

TOKEN=$(curl -s "$BASE_URL/api/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}" \
  | jq -r '.token')

send_telemetry() {
  local parameter_type="$1"
  local value="$2"
  local unit="$3"
  local source_id="$4"
  local measured_at

  measured_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

  curl -s "$BASE_URL/api/telemetry" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
      \"parameterType\":\"$parameter_type\",
      \"value\":$value,
      \"unit\":\"$unit\",
      \"sourceId\":\"$source_id\",
      \"measuredAt\":\"$measured_at\"
    }" > /dev/null
}

case "${1:-}" in
  normal)
    send_telemetry pressure 95 bar demo-normal
    send_telemetry moisture 22 percent demo-normal
    send_telemetry drive_load 45 percent demo-normal
    echo "Normal scenario sent"
    ;;

  warning)
    send_telemetry pressure 88 bar demo-warning
    echo "Warning scenario sent"
    ;;

  critical)
    send_telemetry pressure 145 bar demo-critical
    echo "Critical scenario sent"
    ;;

  anomaly)
    for i in 0 1 2 3 4
    do
      pressure=$((90 + i * 3))
      moisture=$(awk "BEGIN {print 23 - $i * 0.6}")
      drive_load=$((45 + i * 3))

      send_telemetry pressure "$pressure" bar demo-anomaly
      send_telemetry moisture "$moisture" percent demo-anomaly
      send_telemetry drive_load "$drive_load" percent demo-anomaly

      sleep 1
    done

    echo "Anomaly scenario sent"
    ;;

  *)
    echo "Usage: $0 {normal|warning|critical|anomaly}"
    exit 1
    ;;
esac
```

Сделать файл исполняемым:

```bash
chmod +x scripts/demo.sh
```

---

## Симулятор телеметрии

Если включён симулятор, он может отправлять данные автоматически.

Пример переменных:

```env
SIMULATOR_MODE=normal
SIMULATOR_INTERVAL=2s
SIMULATOR_SOURCE_ID=http-simulator
```

Возможные режимы зависят от реализации симулятора проекта.

---

## Основная логика расчёта индекса качества

Индекс качества начинается со значения `100`.

На него влияют:

- активные warning-события;
- активные critical-события;
- обнаруженные аномалии;
- веса параметров.

Чем серьёзнее отклонение, тем сильнее снижается индекс.

Пример:

```text
нет событий и аномалий → индекс около 100
warning-событие → индекс снижается умеренно
critical-событие → индекс снижается сильнее
комбинированная аномалия → индекс дополнительно снижается
```

---

## Полезные команды

Запуск:

```bash
make compose-up
```

Остановка:

```bash
make compose-down
```

Тесты:

```bash
go test ./...
```

Покрытие:

```bash
go test ./... -cover
```

HTML-отчёт покрытия:

```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

Просмотр контейнеров:

```bash
docker compose ps
```

Логи backend:

```bash
docker compose logs -f backend
```

Логи Kafka:

```bash
docker compose logs -f kafka
```

Логи PostgreSQL:

```bash
docker compose logs -f postgres
```

---

## Статус проекта

Реализовано:

- приём телеметрии;
- хранение телеметрии;
- история измерений;
- уставки;
- события;
- подтверждение событий;
- аномалии;
- индекс качества;
- веса качества;
- роли пользователей;
- JWT-аутентификация;
- bcrypt-хеширование паролей;
- UI оператора/технолога;
- Docker Compose запуск;
- тесты;
- демонстрационные сценарии.

---

## Видеодемонстрация работы сервиса

https://github.com/user-attachments/assets/bd433cbd-45e3-4d0f-885c-1affd08e9bb3

## Назначение проекта

Проект разработан как информационно-аналитическая система для контроля качества технологического процесса экструзии кормов. Он демонстрирует полный цикл обработки производственных данных: от поступления телеметрии до аналитической оценки состояния процесса и отображения результата пользователю.

![how it really works](docs/assets/demo/dog.png)
