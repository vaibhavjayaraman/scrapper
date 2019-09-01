FROM golang:1.12.0-alpine3.9 as build-env
RUN apk add --update --no-cache ca-certificates git

WORKDIR /src
COPY go.mod .
COPY go.sum .

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o /go/bin/scrapper

FROM scratch
COPY --from=build-env /go/bin/scrapper /go/bin/scrapper
CMD ["/go/bin/scrapper"]
