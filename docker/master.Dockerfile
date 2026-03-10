FROM golang:1.26 AS build
WORKDIR /app
COPY . .
RUN go build -o /out/master ./cmd/master

FROM gcr.io/distroless/base-debian12
COPY --from=build /out/master /master
EXPOSE 50051
ENTRYPOINT ["/master"]