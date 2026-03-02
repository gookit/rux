# Start by building the application.
#
# @build-example docker build . -f gcr.Dockerfile -t gcr:test
#
FROM golang:1.14 as build

ENV APP_DIR $GOPATH/src/appointment
RUN mkdir -p $APP_DIR
#WORKDIR /go/src/app
WORKDIR $APP_DIR

COPY . .

RUN go build -ldflags '-w -s' -o /go/bin/app

# Now copy it into our base image.
#FROM gcr.io/distroless/base
FROM myaniu/gcr.io-distroless-base
COPY --from=build /go/bin/app /
ENTRYPOINT ["/app"]
