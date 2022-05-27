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
ARG UID=1000

RUN adduser \
    --disabled-password \
    --no-create-home \
    --shell /docker-app \
    --gecos "" \
    --uid ${UID} \
    --home / \
    app

FROM scratch

ARG OS=linux_amd64

COPY ./cmd/accural/accural_$OS /docker-app
COPY --from=build /etc/passwd /etc/passwd
USER app

ENV SERVER_ADDRESS 0.0.0.0:8080
EXPOSE 8080

ENTRYPOINT ["/docker-app"]
