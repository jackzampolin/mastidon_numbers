FROM alpine:latest

COPY ./mastidon /usr/bin/mastidon

RUN apk --no-cache add ca-certificates && update-ca-certificates

CMD ["mastidon"]