FROM golang:1.16

WORKDIR /go/src/gdrive-webdav
COPY . .
RUN go mod tidy && go build .

EXPOSE 8765
ENTRYPOINT ["./gdrive-webdav" ]
