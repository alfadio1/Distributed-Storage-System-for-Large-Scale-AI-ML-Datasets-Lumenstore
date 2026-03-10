FROM golang:1.26 AS build
WORKDIR /app
COPY . .
RUN go build -o /out/node ./cmd/node

FROM gcr.io/distroless/base-debian12
COPY --from=build /out/node /node
EXPOSE 6001
ENTRYPOINT ["/node"]