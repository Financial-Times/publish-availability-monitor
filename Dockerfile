FROM alpine

ADD *.go /publish-availability-monitor/
ADD content/*.go /publish-availability-monitor/content/
ADD config.json.template /publish-availability-monitor/config.json
ADD startup.sh /

RUN apk add --update bash \
  && apk --update add git bzr \
  && echo "http://dl-4.alpinelinux.org/alpine/edge/community" >> /etc/apk/repositories \
  && apk --update add go \
  && export GOPATH=/gopath \
  && REPO_PATH="github.com/Financial-Times/publish-availability-monitor" \
  && mkdir -p $GOPATH/src/${REPO_PATH} \
  && mv /publish-availability-monitor/* $GOPATH/src/${REPO_PATH} \
  && cd $GOPATH/src/${REPO_PATH} \
  && go get \
  && go test ./... \
  && go build \
  && mv publish-availability-monitor /app \
  && mv config.json /config.json \
  && apk del go git bzr \
  && rm -rf $GOPATH /var/cache/apk/*

CMD [ "/startup.sh" ]
