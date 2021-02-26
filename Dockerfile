FROM golang:1.16-alpine AS build-env

WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN apk add --no-cache upx && \
    go version && \
    go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags '-w -s' -o /go/bin/longcat && \
    upx /go/bin/longcat
FROM scratch
COPY --from=build-env /go/bin/longcat /go/bin/longcat
ENTRYPOINT ["/go/bin/longcat"]
