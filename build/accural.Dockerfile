FROM golang:alpine3.15 AS build
#
#WORKDIR /app
#COPY go.mod ./
#COPY go.sum ./
#
#RUN go mod download
#COPY ./ ./
#
#RUN CGO_ENABLED=0 go build -o /docker-app cmd/gophermart/main.go
#
ARG OS=linux_amd64
COPY ./cmd/accrual/accrual_${OS} /docker-app

ARG UID=1000

RUN adduser \
    --disabled-password \
    --no-create-home \
    --shell /docker-app \
    --gecos "" \
    --uid ${UID} \
    --home / \
    app ; \
    chmod a+rx /docker-app

#FROM golang:alpine3.15
FROM ubuntu:18.04

COPY --from=build /docker-app /docker-app
COPY --from=build /etc/passwd /etc/passwd
USER app

ENV ACCRUAL_SYSTEM_ADDRESS 0.0.0.0:8080
EXPOSE 8080

CMD ["/docker-app"]
