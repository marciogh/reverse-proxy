FROM golang:alpine
ADD reverse-proxy /
CMD ["/reverse-proxy"]
