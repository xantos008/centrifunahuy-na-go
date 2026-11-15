# Modified by the Cedrazeka authors, 2026. See NOTICE.
FROM alpine:3.24

ARG USER=cedrazeka
ARG UID=1000
ARG GID=1000

RUN addgroup -S -g $GID $USER && \
    adduser -S -G $USER -u $UID $USER

RUN apk --no-cache upgrade && \
    apk --no-cache add ca-certificates && \
    update-ca-certificates

USER $USER

WORKDIR /cedrazeka

COPY cedrazeka /usr/local/bin/cedrazeka

CMD ["cedrazeka"]
