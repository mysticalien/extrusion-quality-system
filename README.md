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


## Авторизация

В системе используется JWT-аутентификация.

Схема работы:

```text
login + password -> проверка bcrypt-хэша -> выдача JWT -> запросы с Bearer token
```

Пароли пользователей не хранятся в базе данных в открытом виде.  
В таблицу `users` записывается только `password_hash`, созданный через bcrypt.

После успешного входа backend возвращает JWT-токен:

```bash
POST /api/login
```

Пример ответа:

```json
{
  "token": "<jwt-token>",
  "user": {
    "id": 1,
    "username": "operator.user",
    "role": "operator",
    "isActive": true
  }
}
```

Для обращения к защищённым API нужно передавать токен в заголовке:

```bash
Authorization: Bearer <jwt-token>
```

Без токена backend возвращает `401 Unauthorized`.  
Если токен есть, но у роли недостаточно прав, backend возвращает `403 Forbidden`.

## Роли пользователей

В системе используются три роли:

```text
operator
technologist
admin
```

Права ролей:

```text
operator:
- просмотр dashboard;
- просмотр текущих параметров;
- просмотр активных событий;
- подтверждение событий;
- просмотр текущего индекса качества.

technologist:
- всё, что может operator;
- просмотр истории параметров;
- просмотр аномалий;
- изменение уставок.

admin:
- управление пользователями;
- создание пользователей;
- изменение ролей;
- активация и деактивация пользователей;
- сброс паролей.
```

## Инициализация тестовых пользователей

Перед запуском авторизации нужно создать пользователей в таблице `users`.

Имена пользователей можно задать через переменные окружения:

```bash
export APP_OPERATOR_USERNAME="operator.user"
export APP_TECHNOLOGIST_USERNAME="technologist.user"
export APP_ADMIN_USERNAME="admin.user"
```

Пароли лучше вводить скрыто, чтобы они не попали в историю команд:

```bash
read -s -p "Operator password: " APP_OPERATOR_PASSWORD
echo

read -s -p "Technologist password: " APP_TECHNOLOGIST_PASSWORD
echo

read -s -p "Admin password: " APP_ADMIN_PASSWORD
echo
```

Сгенерировать SQL с bcrypt-хэшами:

```bash
cat > /tmp/generate_users_sql.go <<'EOF'
package main

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type userSeed struct {
	Username string
	Password string
	Role     string
}

func quote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func requireEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		panic("missing required environment variable: " + name)
	}

	return value
}

func main() {
	users := []userSeed{
		{
			Username: requireEnv("APP_OPERATOR_USERNAME"),
			Password: requireEnv("APP_OPERATOR_PASSWORD"),
			Role:     "operator",
		},
		{
			Username: requireEnv("APP_TECHNOLOGIST_USERNAME"),
			Password: requireEnv("APP_TECHNOLOGIST_PASSWORD"),
			Role:     "technologist",
		},
		{
			Username: requireEnv("APP_ADMIN_USERNAME"),
			Password: requireEnv("APP_ADMIN_PASSWORD"),
			Role:     "admin",
		},
	}

	for _, user := range users {
		hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			panic(err)
		}

		fmt.Printf(`
INSERT INTO users (
	username,
	password_hash,
	role,
	is_active,
	created_at,
	updated_at
)
VALUES (
	%s,
	%s,
	%s,
	true,
	now(),
	now()
)
ON CONFLICT (username)
DO UPDATE SET
	password_hash = EXCLUDED.password_hash,
	role = EXCLUDED.role,
	is_active = true,
	updated_at = now();
`,
			quote(user.Username),
			quote(string(hash)),
			quote(user.Role),
		)
	}
}
EOF

go run /tmp/generate_users_sql.go > /tmp/seed_users.sql
```

Применить SQL к PostgreSQL в Docker:

```bash
docker exec -i extrusion_postgres sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"' < /tmp/seed_users.sql
```

Проверить пользователей:

```bash
docker exec -it extrusion_postgres sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT id, username, role, is_active, length(password_hash) AS password_hash_length FROM users ORDER BY id;"'
```

Ожидаемо длина bcrypt-хэша должна быть `60`.

## Проверка входа

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d "{
    \"username\": \"$APP_OPERATOR_USERNAME\",
    \"password\": \"$APP_OPERATOR_PASSWORD\"
  }" \
  | python3 -c 'import json,sys; print(json.load(sys.stdin)["token"])')

curl -i http://localhost:8080/api/me \
  -H "Authorization: Bearer $TOKEN"
```
