FROM golang

WORKDIR /easy-rch-printf-print-server
COPY . .

RUN go build ./src/main.go

CMD ["/easy-rch-printf-print-server/main"]
