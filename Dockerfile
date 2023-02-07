FROM golang:alpine AS builder

WORKDIR /builder/

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN go build -o passphrasebot .

FROM alpine
WORKDIR /app/
COPY --from=builder /builder/passphrasebot .

RUN mkdir /secret/

# Run with
# docker run --mount type=bind,source=/etc/secret/,target=/secret/,readonly 123
# Optional parameter, this .env file will be used only if --mount is not specified.
# docker run --mount type=bind,source=/etc/secret/,target=/secret/,readonly -d --name container-name 123
# if you have .env file in the /etc/secret on the docker host. Otherwise
# docker run -d --name container-name 123
COPY --from=builder /builder/.env /secret/

WORKDIR /app/

CMD [ "./passphrasebot" ]