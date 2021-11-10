FROM golang:1.17.3-alpine3.14 AS build
WORKDIR /go/src/docker-plugin-vault
COPY go.mod go.sum main.go ./
RUN go mod download \
  && CGO_ENABLED=0 go install -v

FROM scratch
COPY --from=build "/go/bin/docker-plugin-vault" "/go/bin/docker-plugin-vault"
ENTRYPOINT ["/go/bin/docker-plugin-vault"]
