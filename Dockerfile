FROM golang:alpine AS builder

RUN mkdir -p /app/

COPY . /app/

WORKDIR /app/

RUN go build -o passphrasebot .

FROM alpine
COPY --from=builder /app/passphrasebot /app/

RUN mkdir /secret/

# Run with
# docker run --mount type=bind,source=/etc/secret/,target=/secret/,readonly 123
# Optional parameter, this .env file will be used only if --mount is not specified.
# docker run --mount type=bind,source=/etc/secret/,target=/secret/,readonly -d --name container-name 123
# if you have .env file in the /etc/secret on the docker host. Otherwise
# docker run -d --name container-name 123
COPY --from=builder /app/.env /secret/

WORKDIR /app/

CMD [ "./passphrasebot" ]