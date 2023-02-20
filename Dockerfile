FROM golang AS builder

WORKDIR /src
COPY . .
ENV CGO_ENABLED=0
RUN go build -v

FROM alpine

EXPOSE 2112/tcp

COPY --from=builder /src/upsprom /bin/upsprom

ENTRYPOINT ["/bin/upsprom"]
