FROM golang:1.18.1-alpine AS builder

WORKDIR /src

#RUN mkdir /user && \
#    echo 'nobody:x:65534:65534:nobody:/:' > /user/passwd && \
#    echo 'nobody:x:65534:' > /user/group

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -o /app .


FROM alpine AS final

WORKDIR /src

COPY --from=builder /app ./app

EXPOSE 1110
# Run the compiled binary.
ENTRYPOINT ["./app"]

