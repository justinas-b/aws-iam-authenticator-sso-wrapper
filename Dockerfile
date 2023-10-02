FROM golang:1.21-alpine
RUN apk update && apk upgrade

RUN mkdir -p /app
WORKDIR /app
COPY aws-iam-authenticator-sso-wrapper .

CMD ["./aws-iam-authenticator-sso-wrapper"]