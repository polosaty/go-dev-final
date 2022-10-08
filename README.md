# go-musthave-diploma-tpl

Индивидуальный дипломный проекта курса «Go-разработчик»

## Обновление swagger
```shell
go install github.com/swaggo/swag/cmd/swag@latest
swag init -g handlers\handlers.go  --parseDependency
```

## Запуск пиложения
```shell
docker-compose -f deploy/docker-compose.yml build
docker-compose -f deploy/docker-compose.yml up
```
