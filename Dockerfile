FROM golang
WORKDIR /go/src/lks-jabar-api
COPY . .

# Download all the dependencies
RUN go get -d -v ./...

# Install the package
RUN go install -v ./...
RUN go build
CMD ["./lks-jabar-api"]