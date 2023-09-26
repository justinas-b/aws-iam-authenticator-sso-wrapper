FROM alpine:3.15.0
COPY aws-iam-authenticator-sso-wrapper /usr/bin/aws-iam-authenticator-sso-wrapper
ENTRYPOINT [ "/usr/bin/aws-iam-authenticator-sso-wrapper" ]
