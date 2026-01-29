# GophKeeper

Клиент-серверная система для безопасного хранения логинов, паролей, бинарных данных и другой приватной информации.

## Возможности

- **Безопасное хранение** логинов/паролей, текстовых заметок, файлов, банковских карт
- **End-to-End шифрование** — данные шифруются на клиенте, сервер не имеет доступа к открытым данным
- **Синхронизация** между устройствами
- **Кросс-платформенность** — клиент для Windows, Linux, macOS (билды через отдельные команды)
- **Офлайн-доступ** — локальное хранилище на клиенте

## Архитектура

```
┌──────────────┐          gRPC/TLS          ┌──────────────┐
│    Client    │ ◄───────────────────────► │    Server    │
│   (SQLite)   │                            │ (PostgreSQL) │
└──────────────┘                            └──────────────┘
      │                                            │
      │ AES-256-GCM                                │ bcrypt
      │ PBKDF2                                     │ JWT
      ▼                                            ▼
┌──────────────┐                            ┌──────────────┐
│ Зашифрованные│                            │ Зашифрованные│
│    данные    │                            │    данные    │
└──────────────┘                            └──────────────┘
```

## Быстрый старт

### Требования

- Go 1.21+
- PostgreSQL 14+
- protoc (для генерации gRPC кода)

### Сервер

```bash
# Применить миграции
migrate -path ./migrations -database "postgres://user:pass@localhost:5432/gophkeeper?sslmode=disable" up

# Запустить сервер
go run ./cmd/server \
  --address=:3200 \
  --database-dsn="postgres://user:pass@localhost:5432/gophkeeper?sslmode=disable" \
  --jwt-secret="your-secret-key-min-32-chars!!!!" \
  --log-level=info
```
Или через переменные окружения:

```bash
export GOPHKEEPER_ADDRESS=":3200"
export GOPHKEEPER_DATABASE_DSN="postgres://user:pass@localhost:5432/gophkeeper?sslmode=disable"
export GOPHKEEPER_JWT_SECRET="your-secret-key-min-32-chars!!!!"
export GOPHKEEPER_LOG_LEVEL="info"

go run ./cmd/server
```

### Клиент

```bash
# Сборка
go build -o gophkeeper ./cmd/client

# Регистрация
./gophkeeper register -l myuser -p mypassword -m masterpassword

# Вход
./gophkeeper login -l myuser -p mypassword -m masterpassword

# Добавить учётные данные
./gophkeeper credentials add -n "GitHub" -l user@mail.com -p secret123 -u github.com -m masterpassword

# Просмотреть список
./gophkeeper credentials list -m masterpassword

# Получить данные
./gophkeeper credentials get "GitHub" -m masterpassword

# Синхронизация
./gophkeeper sync -m masterpassword

# Выход
./gophkeeper logout
```

## CLI команды

```
gophkeeper
├── register              # Регистрация нового пользователя
├── login                 # Вход в систему
├── logout                # Выход из системы
├── sync                  # Синхронизация с сервером
├── version               # Информация о версии
│
├── credentials (cred)    # Логины/пароли
│   ├── add               # Добавить
│   ├── list              # Список
│   ├── get <name>        # Получить
│   └── delete <name>     # Удалить
│
├── text                  # Текстовые заметки
│   ├── add / list / get / delete
│
├── binary (bin)          # Бинарные файлы
│   ├── add <file>        # Добавить файл
│   ├── list              # Список
│   ├── get <name> -o     # Скачать
│   └── delete <name>     # Удалить
│
└── card                  # Банковские карты
    ├── add / list / get / delete
```

## Безопасность

### Шифрование

- **Алгоритм**: AES-256-GCM
- **Деривация ключа**: PBKDF2-SHA256 (100,000 итераций)
- **Хранение паролей**: bcrypt (cost=12)

### E2E шифрование

Все данные шифруются на клиенте **до** отправки на сервер. Сервер хранит только зашифрованные данные и не имеет возможности их расшифровать.

```
Master Password → PBKDF2 → Master Key → шифрует Data Key
                                              ↓
                                   Data Key → шифрует данные
```

### Передача данных

- gRPC с поддержкой TLS
- JWT токены для авторизации (access + refresh)

## Разработка

### Структура проекта

```
gophkeeper/
├── cmd/
│   ├── client/           # CLI-клиент
│   └── server/           # Сервер
├── internal/
│   ├── client/           # Логика клиента
│   │   ├── api/          # gRPC клиент
│   │   ├── crypto/       # E2E шифрование
│   │   ├── service/      # Бизнес-логика
│   │   └── storage/      # SQLite хранилище
│   ├── model/            # Доменные модели
│   └── server/           # Логика сервера
│       ├── grpc/         # gRPC обработчики
│       ├── repository/   # PostgreSQL репозитории
│       └── service/      # Бизнес-логика
├── api/proto/            # Protobuf схемы
├── migrations/           # SQL миграции
└── pkg/                  # Общие пакеты
    ├── crypto/           # Криптография
    ├── jwt/              # JWT токены
    └── logger/           # Логирование (zap)
```

### Сборка

```bash
# Установка зависимостей
go mod download

# Генерация protobuf
task proto-gen

# Сборка
task build

# Тесты
task test

# Тесты с покрытием
task test-cover

# Кросс-компиляция
task build-all
```

### Тестирование

```bash
# Все тесты
go test ./...

# С покрытием
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

Текущее покрытие тестами: **>70%**

## Конфигурация

### Сервер

| Параметр          | Флаг                  | Env                       | По умолчанию |
| ----------------- | --------------------- | ------------------------- | ------------ |
| Адрес             | `--address`           | `GOPHKEEPER_ADDRESS`      | `:3200`      |
| База данных       | `--database-dsn`      | `GOPHKEEPER_DATABASE_DSN` | -            |
| JWT секрет        | `--jwt-secret`        | `GOPHKEEPER_JWT_SECRET`   | -            |
| Access Token TTL  | `--access-token-ttl`  | -                         | `15m`        |
| Refresh Token TTL | `--refresh-token-ttl` | -                         | `168h`       |
| TLS сертификат    | `--tls-cert`          | `GOPHKEEPER_TLS_CERT`     | -            |
| TLS ключ          | `--tls-key`           | `GOPHKEEPER_TLS_KEY`      | -            |
| Уровень логов     | `--log-level`         | `GOPHKEEPER_LOG_LEVEL`    | `info`       |

### Клиент

| Параметр          | Env                         | По умолчанию     |
| ----------------- | --------------------------- | ---------------- |
| Адрес сервера     | `GOPHKEEPER_SERVER_ADDRESS` | `localhost:3200` |
| Директория данных | `GOPHKEEPER_DATA_DIR`       | `~/.gophkeeper`  |
| CA сертификат     | `GOPHKEEPER_TLS_CERT`       | -                |

## Лицензия

MIT
