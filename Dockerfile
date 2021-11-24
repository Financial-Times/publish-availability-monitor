FROM golang:1

ENV PROJECT=publish-availability-monitor

ENV ORG_PATH="github.com/Financial-Times"
ENV BUILDINFO_PACKAGE="github.com/Financial-Times/service-status-go/buildinfo."

COPY . /${PROJECT}/
WORKDIR /${PROJECT}

RUN VERSION="version=$(git describe --tag --always 2> /dev/null)" \
  && DATETIME="dateTime=$(date -u +%Y%m%d%H%M%S)" \
  && REPOSITORY="repository=$(git config --get remote.origin.url)" \
  && REVISION="revision=$(git rev-parse HEAD)" \
  && BUILDER="builder=$(go version)" \
  && LDFLAGS="-s -w -X '"${BUILDINFO_PACKAGE}$VERSION"' -X '"${BUILDINFO_PACKAGE}$DATETIME"' -X '"${BUILDINFO_PACKAGE}$REPOSITORY"' -X '"${BUILDINFO_PACKAGE}$REVISION"' -X '"${BUILDINFO_PACKAGE}$BUILDER"'" \
  && CGO_ENABLED=0 go build -mod=readonly -o /artifacts/${PROJECT} -v -ldflags="${LDFLAGS}" \
  && echo "Build flags: $LDFLAGS" 

# Multi-stage build - copy certs and the binary into the image
# We build from alpine because scratch does not have bash
FROM alpine
WORKDIR /	
COPY --from=0 /publish-availability-monitor/config.json.template /config.json 
COPY --from=0 /publish-availability-monitor/startup.sh /
COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=0 /artifacts/* /

CMD [ "/startup.sh" ]
