FROM golang:alpine AS engine_invoice

WORKDIR /appbuilds

COPY . .

RUN go mod tidy
RUN go build -o binary


FROM alpine:latest as enginesavetransaksirelease
WORKDIR /app
RUN apk add tzdata
COPY --from=engine_invoice /appbuilds/binary .
COPY --from=engine_invoice /appbuilds/.env /app/.env
ENV DB_USER="admindb"
ENV DB_PASS="asd123QWE"
ENV DB_HOST="128.199.124.131"
ENV DB_PORT="5432"
ENV DB_NAME="admindb"
ENV DB_SCHEMA="db_wigo"
ENV DB_DRIVER="postgres"
ENV DB_REDIS_HOST="128.199.124.131"
ENV DB_REDIS_PORT="6379"
ENV DB_REDIS_PASSWORD="asdQWE123!@#"
ENV DB_REDIS_NAME="2"
ENV DB_CONF_COMPANY="NUKE"
ENV DB_CONF_CURR="IDR"
ENV TZ=Asia/Jakarta

RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

ENTRYPOINT [ "./binary" ]