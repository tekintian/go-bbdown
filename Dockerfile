FROM alpine:latest
RUN apk add --no-cache ffmpeg aria2
COPY bbdown /usr/local/bin/
ENTRYPOINT ["bbdown"]
