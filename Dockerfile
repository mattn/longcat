FROM golang:alpine AS build-env

WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /go/bin/longcat
FROM scratch
COPY --from=build-env /go/bin/longcat /go/bin/longcat
ENTRYPOINT ["/go/bin/longcat"]
