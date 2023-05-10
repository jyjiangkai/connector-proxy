FROM --platform=amd64 golang:1.18 as builder
WORKDIR /workspace

COPY . .
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod download

ARG TARGETOS
ARG TARGETARCH
RUN mkdir bin
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/proxy main.go

FROM --platform=amd64 ubuntu:22.10
WORKDIR /vanus
COPY --from=builder /workspace/bin/proxy bin/proxy
ENTRYPOINT ["bin/proxy"]

