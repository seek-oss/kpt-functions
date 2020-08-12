FROM golang:1.14
ENV CGO_ENABLED=0
ARG FUNCTION
COPY . /src
WORKDIR /src
RUN go build -v -o /usr/local/bin/fn cmd/${FUNCTION}/*.go

FROM alpine:latest
COPY --from=0 /usr/local/bin/fn /usr/local/bin/fn
CMD ["/usr/local/bin/func"]
