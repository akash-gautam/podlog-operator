FROM alpine:3.8

ENV OPERATOR=/usr/local/bin/podlog-operator \
    USER_UID=1001 \
    USER_NAME=podlog-operator

# install operator binary
COPY _output/bin/podlog-operator ${OPERATOR}
COPY bin /usr/local/bin
RUN  /usr/local/bin/user_setup
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
