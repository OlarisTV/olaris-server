FROM ubuntu:xenial as base

FROM base as ffmpeg-build

WORKDIR     /tmp/workdir

RUN     apt-get -y update && \
        apt-get install -y --no-install-recommends ca-certificates expat libgomp1 git build-essential

#RUN     apt-get -y build-dep ffmpeg

# Roughly apt-get build-dep ffmpeg
RUN     apt-get install -y --no-install-recommends flite1-dev frei0r-plugins-dev ladspa-sdk libass-dev libbluray-dev libbs2b-dev libbz2-dev libcaca-dev libcdio-paranoia-dev libcrystalhd-dev libdc1394-22-dev libfontconfig1-dev libfreetype6-dev libfribidi-dev libgl1-mesa-dev libgme-dev libgnutls-dev libgsm1-dev libiec61883-dev libavc1394-dev libjack-jackd2-dev liblzma-dev libmodplug-dev libmp3lame-dev libopenal-dev libopencore-amrnb-dev libopencore-amrwb-dev libopenjpeg-dev libopus-dev libpulse-dev librtmp-dev libsctp-dev libsdl-dev libshine-dev libsnappy-dev libsoxr-dev libspeex-dev libssh-gcrypt-dev libtheora-dev libtwolame-dev libvdpau-dev libvo-aacenc-dev libvo-amrwbenc-dev libvorbis-dev libvpx-dev libwavpack-dev libwebp-dev libx264-dev yasm libopenjp2-7-dev libx265-dev libxvidcore-dev libzmq3-dev libzvbi-dev libxml2-dev libomxil-bellagio-dev

RUN apt-get autoremove -y && \
        apt-get clean -y

RUN git clone -b master https://gitlab.com/olaris/ffmpeg.git

ARG        PREFIX=/opt/ffmpeg
ARG        MAKEFLAGS="-j4"

RUN \
        cd ffmpeg && \
        ./configure --prefix="${PREFIX}" \
        --enable-gpl \
        --disable-stripping \
        --enable-avresample \
        --enable-avisynth \
        --enable-gnutls \
        --enable-ladspa \
        --enable-libass \
        --enable-libbluray \
        --enable-libbs2b \
        --enable-libcaca \
        --enable-libcdio \
        --enable-libflite \
        --enable-libfontconfig \
        --enable-libfreetype \
        --enable-libfribidi \
        --enable-libgme \
        --enable-libgsm \
        --enable-libmp3lame \
        --enable-libopus \
        --enable-libpulse \
        --enable-libshine \
        --enable-libsnappy \
        --enable-libsoxr \
        --enable-libspeex \
        --enable-libssh \
        --enable-libtheora \
        --enable-libtwolame \
        --enable-libvorbis \
        --enable-libvpx \
        --enable-libwavpack \
        --enable-libwebp \
        --enable-libx265 \
        --enable-libxml2 \
        --enable-libxvid \
        --enable-libzmq \
        --enable-libzvbi \
        --enable-openal \
        --enable-opengl \
        --enable-libdc1394 \
        --enable-libiec61883 \
        --enable-frei0r \
        --enable-libx264 \
        --enable-libx265 \
        --enable-static && \
        make && \
        make install

FROM        base AS ffmpeg-release

COPY --from=ffmpeg-build /opt/ffmpeg /opt/ffmpeg
env PATH /opt/ffmpeg/bin:$PATH

# Install ffmpeg dependencies
RUN     apt-get -y update && \
        apt-get -y --no-install-recommends install libaacs0 libasound2 libasound2-data libass5 libasyncns0 libavc1394-0 libavcodec-ffmpeg56 libavdevice-ffmpeg56 libavfilter-ffmpeg5 libavformat-ffmpeg56 libavresample-ffmpeg2 libavutil-ffmpeg54 libbdplus0 libbluray1 libbs2b0 libcaca0 libcdio-cdda1 libcdio-paranoia1 libcdio13 libcrystalhd3 libdc1394-22 libelf1 libflac8 libflite1 libfontconfig1 libfreetype6 libfribidi0 libgl1-mesa-dri libgl1-mesa-glx libglapi-mesa libgme0 libgomp1 libgraphite2-3 libgsm1 libharfbuzz0b libicu55 libiec61883-0 libjack-jackd2-0 libjson-c2 libllvm5.0 libmodplug1 libmp3lame0 libnuma1 libogg0 libopenal-data libopenal1 libopencv-core2.4v5 libopencv-imgproc2.4v5 libopenjpeg5 libopus0 liborc-0.4-0 libpciaccess0 libpng12-0 libpostproc-ffmpeg53 libpulse0 libraw1394-11 libsamplerate0 libschroedinger-1.0-0 libsdl1.2debian libsensors4 libshine3 libslang2 libsnappy1v5 libsndfile1 libsodium18 libsoxr0 libspeex1 libssh-gcrypt-4 libswresample-ffmpeg1 libswscale-ffmpeg3 libtbb2 libtheora0 libtwolame0 libtxc-dxtn-s2tc0 libusb-1.0-0 libva1 libvdpau1 libvorbis0a libvorbisenc2 libvpx3 libwavpack1 libwebp5 libx11-6 libx11-data libx11-xcb1 libx264-148 libx265-79 libxau6 libxcb-dri2-0 libxcb-dri3-0 libxcb-glx0 libxcb-present0 libxcb-shape0 libxcb-shm0 libxcb-sync1 libxcb-xfixes0 libxcb1 libxdamage1 libxdmcp6 libxext6 libxfixes3 libxml2 libxshmfence1 libxv1 libxvidcore4 libxxf86vm1 libzmq5 libzvbi-common libzvbi0 sgml-base ucf xml-core && \
        apt-get autoremove -y && apt-get clean -y

# To test
RUN ffmpeg --help

RUN apt-get -y install software-properties-common
RUN add-apt-repository ppa:gophers/archive
RUN apt-get update
RUN apt-get -y install golang-1.10-go git

ENV GOPATH="/go"
ENV PATH="/usr/lib/go-1.10/bin:${GOPATH}/bin:${PATH}"

ADD . /go/src/gitlab.com/olaris/olaris-server
RUN mkdir /var/media

WORKDIR /go/src/gitlab.com/olaris/olaris-server

RUN go get github.com/jteeuwen/go-bindata/...
RUN go get github.com/elazarl/go-bindata-assetfs/...
RUN apt-get install -y curl apt-transport-https && curl -sL https://deb.nodesource.com/setup_8.x | bash -&&  curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | apt-key add - && echo "deb https://dl.yarnpkg.com/debian/ stable main" | tee /etc/apt/sources.list.d/yarn.list && \
    apt-get update && apt-get install nodejs yarn -y
RUN make

RUN go get github.com/oxequa/realize

EXPOSE 8080

ENV LOGTOSTDERR=1
ENV V=4
ENTRYPOINT realize start --generate
