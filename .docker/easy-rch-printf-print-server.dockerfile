FROM golang as build

WORKDIR /easy-rch-printf-print-server
COPY . .

RUN go build ./src/main.go

CMD ["/easy-rch-printf-print-server/main"]

FROM alpine as production

COPY --from=build /easy-rch-printf-print-server/main /easy-rch-printf-print-server

CMD ["/easy-rch-printf-print-server"]
