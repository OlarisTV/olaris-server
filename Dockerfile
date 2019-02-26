FROM ubuntu:xenial as base

FROM base AS olaris-dev

RUN  apt-get -y update

RUN apt-get -y install golang-1.10-go git

ENV GOPATH="/go"
ENV PATH="/usr/lib/go-1.10/bin:${GOPATH}/bin:${PATH}"


RUN go get github.com/jteeuwen/go-bindata/...
RUN go get github.com/elazarl/go-bindata-assetfs/...

RUN apt-get install -y curl apt-transport-https gnupg && curl -sL https://deb.nodesource.com/setup_8.x | bash -&&  curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | apt-key add - && echo "deb https://dl.yarnpkg.com/debian/ stable main" | tee /etc/apt/sources.list.d/yarn.list && \
    apt-get update && apt-get install nodejs yarn make -y
RUN go get github.com/cortesi/modd/cmd/modd

ADD . /go/src/gitlab.com/olaris/olaris-server
WORKDIR /go/src/gitlab.com/olaris/olaris-server

RUN mkdir /var/media

EXPOSE 8080

ENTRYPOINT ["/bin/bash", "-c"]

# Remove downloaded archive files
RUN apt-get autoremove -y && apt-get clean -y
