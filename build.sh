CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o reverse-proxy .
docker build . -t docker.atl-paas.net/atlassian/atlassian-reverse-proxy
docker push docker.atl-paas.net/atlassian/atlassian-reverse-proxy

