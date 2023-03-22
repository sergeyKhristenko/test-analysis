FROM alpine:latest

RUN apk update && apk add --no-cache libxml2-utils

COPY parse-test-reports /usr/local/bin/parse-test-reports
RUN chmod +x /usr/local/bin/parse-test-reports

ENTRYPOINT ["/usr/local/bin/parse-test-reports"]