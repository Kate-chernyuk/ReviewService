Для запуска сервиса достаточно использовать Docker

Dockerfile:
```
FROM golang:1.25-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main .

EXPOSE 8080

CMD ["./main"]
```

docker-compose.yml:
```
version: '3.8'

services:
  postgres:
    image: postgres:13-alpine
    environment:
      POSTGRES_DB: pr_review_service
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5

  app:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      DB_HOST: postgres
      DB_PORT: 5432
      DB_USER: postgres
      DB_PASSWORD: password
      DB_NAME: pr_review_service
      DB_SSLMODE: disable
    depends_on:
      postgres:
        condition: service_healthy

volumes:
  postgres_data:
```

Эти файлы уже лежат в корневом репозитории, соответственно, достаточно клонировать проект, а после запустить сборку.

Для сборки и запуска необходимо использовать команду:
```
docker-compose up --build
```

Сервис будет доступен по адресу http://localhost:8080
