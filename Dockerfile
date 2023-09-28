FROM alpine:3.18.3
COPY aws-iam-authenticator-sso-wrapper /usr/bin/aws-iam-authenticator-sso-wrapper
ENTRYPOINT [ "/usr/bin/aws-iam-authenticator-sso-wrapper" ]
