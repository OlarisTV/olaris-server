FROM debian:testing-slim as base

FROM base as ffchunk-build

WORKDIR     /tmp/workdir

RUN     apt-get -y update && \
        apt-get install -y --no-install-recommends ca-certificates git build-essential

RUN apt-get -y --no-install-recommends install \
	    cmake pkg-config \
	    libavformat-dev libavutil-dev libavcodec-dev libswresample-dev \
	    libprotobuf-c-dev
RUN true
RUN git clone -b master https://gitlab.com/olaris/ffchunk.git
RUN mkdir -p ffchunk/build && cd ffchunk/build && cmake .. && make && make install && cd ../..

FROM        base AS olaris-dev

COPY --from=ffchunk-build /usr/local/bin/ffchunk_transmux /usr/local/bin/ffchunk_transmux
COPY --from=ffchunk-build /usr/local/bin/ffchunk_transcode_video /usr/local/bin/ffchunk_transcode_video
COPY --from=ffchunk-build /usr/local/bin/ffchunk_transcode_audio /usr/local/bin/ffchunk_transcode_audio

RUN     apt-get -y update

# Install ffmpeg
RUN     apt-get -y --no-install-recommends install ffmpeg

# Install ffchunk dependencies
# We install -dev packages because their names remain stable between Ubuntu releases
RUN apt-get -y --no-install-recommends install libavformat-dev libavutil-dev libavcodec-dev libswresample-dev \
	libprotobuf-c1

#RUN apt-get -y install software-properties-common gpg
#RUN add-apt-repository ppa:gophers/archive
#RUN apt-get update
RUN apt-get -y install golang-1.10-go git

ENV GOPATH="/go"
ENV PATH="/usr/lib/go-1.10/bin:${GOPATH}/bin:${PATH}"

ADD . /go/src/gitlab.com/olaris/olaris-server
RUN mkdir /var/media

WORKDIR /go/src/gitlab.com/olaris/olaris-server

RUN go get github.com/jteeuwen/go-bindata/...
RUN go get github.com/elazarl/go-bindata-assetfs/...
RUN apt-get -y install golang-1.10-go 
RUN apt-get install -y curl apt-transport-https gnupg && curl -sL https://deb.nodesource.com/setup_8.x | bash -&&  curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | apt-key add - && echo "deb https://dl.yarnpkg.com/debian/ stable main" | tee /etc/apt/sources.list.d/yarn.list && \
    apt-get update && apt-get install nodejs yarn make -y
RUN make

RUN go get github.com/oxequa/realize

EXPOSE 8080

ENV LOGTOSTDERR=1
ENV V=4
ENTRYPOINT realize start --generate

# Remove downloaded archive files
RUN apt-get autoremove -y && apt-get clean -y
