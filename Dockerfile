FROM golang:1.16
ENV CGO_ENABLED=0
ARG FUNCTION
COPY . /src
WORKDIR /src
RUN go build -v -o /usr/local/bin/fn cmd/${FUNCTION}/*.go

FROM alpine:latest
COPY --from=0 /usr/local/bin/fn /usr/local/bin/fn
COPY ssh/known_hosts /.ssh/known_hosts
CMD ["/usr/local/bin/fn"]
