# syntax=docker/dockerfile:1

FROM golang:latest

LABEL maintainer="Artyom <artyomsh01@yandex.ru>"

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN cd microservice/auth/cmd && go build -o auth

EXPOSE 8081

WORKDIR /app/microservice/auth/cmd

CMD ["./auth"]
