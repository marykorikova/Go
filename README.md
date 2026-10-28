# Trip Service

Это учебный сервис для работы с поездками.

Сервис умеет:

* создавать поездку;
* получать поездку по ID;
* завершать поездку;
* проверять работу сервиса и базы данных.

Данные хранятся в PostgreSQL.

## Запуск окружения

Для работы нужны Docker и `tripgoctl`.

Запуск окружения:

```bash
tripgoctl cluster start
tripgoctl environment start
tripgoctl connect

Применить миграции:

export $(grep -v '^#' .env | xargs)
goose -dir migrations postgres "$DATABASE_URL" up

Посмотреть их состояние:

export $(grep -v '^#' .env | xargs)
goose -dir migrations postgres "$DATABASE_URL" status

Откатить последнюю миграцию:

export $(grep -v '^#' .env | xargs)
goose -dir migrations postgres "$DATABASE_URL" down

В проекте создаются две таблицы:

trips — поездки;

trip_status_history — история изменения статусов.


Запуск сервиса

export $(grep -v '^#' .env | xargs)
go run ./cmd/trip-service

Сервис запускается на порту 8080.

Генерация кода
Часть Go-кода генерируется из OpenAPI-контракта.

go tool oapi-codegen -generate types,chi-server -package api -o internal/generated/api.gen.go contracts/openapi/trip-service.openapi.yaml

Сгенерированный файл:

internal/generated/api.gen.go

Его не нужно изменять вручную.

Переменные окружения
В работе используются:

HTTP_ADDR — адрес HTTP-сервера;

LOG_LEVEL — уровень логирования;

SHUTDOWN_TIMEOUT — время ожидания при остановке;

DATABASE_URL — адрес PostgreSQL;

DATABASE_MAX_CONNS — максимальное число соединений;

DATABASE_MIN_CONNS — минимальное число соединений;

DATABASE_MAX_CONN_LIFETIME — время жизни соединения;

DATABASE_CONNECT_TIMEOUT — таймаут подключения;

DATABASE_QUERY_TIMEOUT — таймаут SQL-запроса.

Пример находится в .env.example.

HTTP-запросы

Проверка работы процесса:

curl -i localhost:8080/health

Проверка доступности базы:

curl -i localhost:8080/ready

Создание поездки:

curl -i -X POST localhost:8080/api/v1/trips \
  -H 'Content-Type: application/json' \
  -d '{
    "user_id": "5cb72c04-7650-45c9-a79b-bcdba0631e0c",
    "driver_id": "8860b315-ec86-42eb-a17c-7c163d721ff5",
    "start_point": {
      "latitude": 59.9398,
      "longitude": 30.3146
    },
    "end_point": {
      "latitude": 59.929,
      "longitude": 30.3626
    },
    "price": 1450
  }'

Получение поездки:

curl -i localhost:8080/api/v1/trips/<trip-id>

Завершение поездки:

curl -i -X POST \
  localhost:8080/api/v1/trips/<trip-id>/finish

Решения
Транзакции
Используется уровень изоляции Read Committed.

Создание поездки состоит из двух запросов:

запись поездки в trips;

запись статуса в trip_status_history.

Оба запроса выполняются в одной транзакции. Если один из них завершится с ошибкой, изменения отменятся через ROLLBACK. Если ошибок нет, выполняется COMMIT.

Транзакция хранится в context. Repository берёт её оттуда. Если транзакции нет, repository использует обычный пул соединений.

Если Do вызывается внутри другого Do, новая транзакция не создаётся.

Один водитель и одна активная поездка
В PostgreSQL создан уникальный индекс:

sql
CREATE UNIQUE INDEX trips_driver_active_idx
ON trips (driver_id)
WHERE status = 'active';
Он запрещает создавать две активные поездки одному водителю.

При нарушении индекса PostgreSQL возвращает ошибку 23505. Сервис преобразует её в ответ 409 driver_busy.

Завершение поездки
Поездка завершается только при статусе active:

WHERE id = $1 AND status = 'active'

Поэтому при двух одновременных запросах поездку завершит только один из них. Второй получит ошибку 409 trip_completed.

Health и Ready
/health проверяет, что процесс сервиса работает. В базу этот запрос не ходит.

/ready проверяет PostgreSQL. Если база остановлена, он возвращает 503.

Остановка сервиса
Сервис обрабатывает SIGINT и SIGTERM.

При остановке он перестаёт принимать новые запросы и ждёт завершения текущих запросов. Время ожидания задаётся через SHUTDOWN_TIMEOUT.

