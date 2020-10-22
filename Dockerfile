FROM golang:1.15.3-alpine3.12@sha256:781f57b1983444fe5fb26f18bf0c5adc0039bda074637c91ec141f8a5c2b2cca

WORKDIR /workspace/source

COPY go.* ./
RUN go mod download

COPY . .

#RUN go test

RUN sed -i 's/zap.NewDevelopment()/zap.NewProduction()/' main.go

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -ldflags '-w -extldflags "-static"'

FROM gcr.io/distroless/base:nonroot@sha256:2261b65122adb19da72084617c03a9084c24b33fcd90edd74739f0fd631f0f60

COPY --from=0 /workspace/source/v1 /usr/local/bin/v1

ENTRYPOINT ["v1"]
