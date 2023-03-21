FROM alpine:latest

RUN apk update && apk add --no-cache libxml2-utils

COPY check-junit-failure /usr/local/bin/check-junit-failure
RUN chmod +x /usr/local/bin/check-junit-failure

ENTRYPOINT ["/usr/local/bin/check-junit-failure"]