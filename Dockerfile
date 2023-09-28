FROM alpine:3.18.3
COPY aws-iam-authenticator-sso-wrapper /usr/bin/aws-iam-authenticator-sso-wrapper
RUN apk update && apk upgrade
ENTRYPOINT [ "/usr/bin/aws-iam-authenticator-sso-wrapper" ]
