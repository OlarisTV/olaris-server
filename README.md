# Olaris Server


## Running with Docker

    docker build -t olaris-server .
    docker run -i -t --publish 8080:8080 -v $(pwd):/go/src/gitlab.com/olaris/olaris-server -v ~/Videos:/var/media -v ~/.config/olaris:/root/.config/olaris -t olaris-server


This mounts your local development directory inside the Docker container, allowing you to make changes to the application without rebuilding the container. The container features auto-reload functionality - just save a file, wait a few seconds and reload in your browser!

Use your own media directory to mount at `/var/media` obviously.

## Running manually

### Build dependencies
  * Install Go
	* go get github.com/jteeuwen/go-bindata/...
	* go get github.com/elazarl/go-bindata-assetfs/...

### Running

	`make run`

## Building

  `make build` to build for your local platform.

  `build-with-react` to build and pull in the latest web-interface.

## Custom ffmpeg

Olaris-server currently requires a patched version of ffmpeg to
work correctly.

	git clone https://gitlab.com/olaris/ffmpeg.git
	cd ffmpeg

On Debian Linux, I have successfully used the following command line to build a working binary

	./configure --prefix=/usr --extra-version=1+b2 --toolchain=hardened --libdir=/usr/lib/x86_64-linux-gnu --incdir=/usr/include/x86_64-linux-gnu --enable-gpl --disable-stripping --enable-avresample --enable-avisynth --enable-gnutls --enable-ladspa --enable-libass --enable-libbluray --enable-libbs2b --enable-libcaca --enable-libcdio --enable-libflite --enable-libfontconfig --enable-libfreetype --enable-libfribidi --enable-libgme --enable-libgsm --enable-libmp3lame --enable-libmysofa --enable-libopenjpeg --enable-libopenmpt --enable-libopus --enable-libpulse --enable-librubberband --enable-librsvg --enable-libshine --enable-libsnappy --enable-libsoxr --enable-libspeex --enable-libssh --enable-libtheora --enable-libtwolame --enable-libvorbis --enable-libvpx --enable-libwavpack --enable-libwebp --enable-libx265 --enable-libxml2 --enable-libxvid --enable-libzmq --enable-libzvbi --enable-omx --enable-openal --enable-opengl --enable-sdl2 --enable-libdc1394 --enable-libdrm --enable-libiec61883 --enable-chromaprint --enable-frei0r --enable-libx264 --enable-static
	make -j4

For macOS, see https://trac.ffmpeg.org/wiki/CompilationGuide/macOS

To make Olaris Server use your binary, put the ffmpeg source directory (which will then contain the binary) in your `PATH`. For development, just do

	export PATH=/path/to/your/ffmpeg:$PATH
