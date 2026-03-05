# CryptoWalleTest

Встраиваемый платежный модуль для криптовалютных операций;
Реализован на **Go** с использованием **PostgreSQL**;

## Описание

**CryptoWalleTest** - серверный модуль вывода криптовалют;
Управляет балансами, обрабатывает заявки на вывод средств;
Гарантирует консистентность при конкурентных запросах;
Начальная валюта - `USDT`, расширяемо на другие;

## Возможности

- **Вывод средств** - создание и отслеживание заявок со статусом `pending`;
- **Подтверждение** - перевод заявки в статус `confirmed`;
- **Леджер** - журнал всех финансовых операций;
- **Идемпотентность** - защита от дублирования через `idempotency_key`;
- **Консистентность** - `SELECT FOR UPDATE` блокирует строку баланса в транзакции;
- **Авторизация** - Bearer-токен из переменных окружения;
- **Валидация** - проверка входных данных на уровне API;

## API

Все запросы требуют заголовок `Authorization: Bearer <token>`;

- `GET /v1/users` - список пользователей (`?limit=&offset=`);
- `GET /v1/users/{id}` - получение данных пользователя;
- `PUT /v1/users/{id}` - обновление имени пользователя;
- `GET /v1/withdrawals` - список заявок (`?limit=&offset=`);
- `POST /v1/withdrawals` - создание заявки на вывод;
- `GET /v1/withdrawals/{id}` - получение информации о заявке;
- `POST /v1/withdrawals/{id}/confirm` - подтверждение заявки;
- `GET /v1/ledger` - журнал операций (`?limit=&offset=`);

## Структура проекта

- `cmd/server/` - точка входа сервера;
- `cmd/client/` - тестовый клиент;
- `internal/database/` - подключение к БД и миграции;
- `internal/model/` - структуры данных;
- `migrations/` - SQL-миграции;
- `configs/` - конфигурация (хост, порт, имя БД);
- `secrets/` - пароли и токены (gitignored);
- `scripts/` - скрипты для Docker и генерации секретов;
- `runtime/` - данные PostgreSQL (gitignored);

## Быстрый старт

1. Сгенерировать секреты:
```sh
./scripts/gen-secrets.sh
```

2. Запустить через Docker Compose:
```sh
docker compose up --build
```

Или локально (требуется запущенный PostgreSQL через docker compose):
```sh
./scripts/run-server.sh
```

## Конфигурация

Docker: `configs/server.env` - хост `postgresql`, порт `5432`;
Локально: `configs/server.local.env` - хост `localhost`, порт `5433`;
Секреты: `secrets/server.env` - `DB_PASSWORD`, `APP_AUTH_TOKEN`;

## CLI-клиент

```sh
./scripts/run-client.sh get-user 1
./scripts/run-client.sh create-withdrawal --amount 50 --destination 0xABC --key my-key
./scripts/run-client.sh confirm-withdrawal 1
./scripts/run-client.sh list-users --limit 10
./scripts/run-client.sh list-withdrawals
./scripts/run-client.sh list-ledger
```

Полный список команд: `./scripts/run-client.sh --help`;

## Сборка и скрипты

- `./scripts/run-server.sh` - запустить сервер локально;
- `./scripts/run-client.sh <command>` - запустить CLI-клиент;
- `./scripts/gen-secrets.sh` - сгенерировать пароли (если отсутствуют);
- `./scripts/docker-build.sh [tag]` - собрать Docker-образ;
- `./scripts/docker-push.sh [tag]` - отправить в GitLab registry;
- `make test` - запустить тесты (требуется Docker);
- `make lint` - проверка кода;

## Тесты

```sh
make test
```

Тесты используют `testcontainers-go` для запуска PostgreSQL в Docker;
Покрытие: успешное создание, ошибка баланса, идемпотентность, конкурентный доступ;

## Консистентность

Двойное списание исключено через PostgreSQL транзакции;
`SELECT v_balance FROM t_user WHERE v_id = $1 FOR UPDATE` блокирует строку баланса;
Конкурентные запросы на один баланс сериализуются на уровне строки;
Уникальный индекс на `v_idempotency_key` предотвращает дублирование заявок;