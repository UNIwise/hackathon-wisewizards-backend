############################
# STEP 1 build base
############################
FROM golang:1.23-alpine3.20 AS builder
WORKDIR /build
RUN apk add --no-cache build-base
COPY ["go.mod", "go.sum", "./"]
RUN go mod download -x
COPY . .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o /build/bin/service main.go

############################
# STEP 2 Finalize image
############################
FROM alpine:3.20 AS image-base
WORKDIR /app
COPY --from=builder /build/bin/service /usr/bin/go-template
ENTRYPOINT [ "go-template" ]
CMD [ "serve" ]