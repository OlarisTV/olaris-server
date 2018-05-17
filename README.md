# Bytesized Streaming Server


## Running with Docker

    docker build -t bytesized-streaming .
    docker run -i -t --publish 8080:8080 -v $(pwd):/go/src/gitlab.com/bytesized/bytesized-streaming -v ~/Videos:/var/media -v ~/.config/bss:/root/.config/bss -t bytesized-streaming


This mounts your local development directory inside the Docker container, allowing you to make changes to the application without rebuilding the container. The container features auto-reload functionality - just save a file, wait a few seconds and reload in your browser!

Use your own media directory to mount at `/var/media` obviously.

## Running manually

### Dependencies

	go get github.com/jteeuwen/go-bindata/...
	go get github.com/elazarl/go-bindata-assetfs/...

### Running

	go generate ./... && go run server/bindata.go server/main.go --media_files_dir ~/Videos

## Building React

  Install prereqs

  `npm install create-react-app`
  `cd server/react && yarn install`
  `yarn start` and `yarn sass:watch`

  Development on http://localhost:3000/ once done you can build it so
  the golang app can serve it with `yarn build` and then restarting the
  go app.

  TODO: There is probably a better way to serve and deal with this


## Custom ffmpeg

Bytesized Streaming Server currently requires a patched version of ffmpeg to
work correctly.

	git clone -b bytesized-streaming https://ndreke.de/~leon/dump/ffmpeg
	cd ffmpeg

On Debian Linux, I have successfully used the following command line to build a working binary

	./configure --prefix=/usr --extra-version=1+b2 --toolchain=hardened --libdir=/usr/lib/x86_64-linux-gnu --incdir=/usr/include/x86_64-linux-gnu --enable-gpl --disable-stripping --enable-avresample --enable-avisynth --enable-gnutls --enable-ladspa --enable-libass --enable-libbluray --enable-libbs2b --enable-libcaca --enable-libcdio --enable-libflite --enable-libfontconfig --enable-libfreetype --enable-libfribidi --enable-libgme --enable-libgsm --enable-libmp3lame --enable-libmysofa --enable-libopenjpeg --enable-libopenmpt --enable-libopus --enable-libpulse --enable-librubberband --enable-librsvg --enable-libshine --enable-libsnappy --enable-libsoxr --enable-libspeex --enable-libssh --enable-libtheora --enable-libtwolame --enable-libvorbis --enable-libvpx --enable-libwavpack --enable-libwebp --enable-libx265 --enable-libxml2 --enable-libxvid --enable-libzmq --enable-libzvbi --enable-omx --enable-openal --enable-opengl --enable-sdl2 --enable-libdc1394 --enable-libdrm --enable-libiec61883 --enable-chromaprint --enable-frei0r --enable-libx264 --enable-static
	make -j4

For macOS, see https://trac.ffmpeg.org/wiki/CompilationGuide/macOS

To make Bytesized Streaming Server use your binary, put the ffmpeg source directory (which will then contain the binary) in your `PATH`. For development, just do

	export PATH=/path/to/your/ffmpeg:$PATH

## Test movies

### Sintel

wget https://download.blender.org/durian/movies/Sintel.2010.720p.mkv && \
wget https://download.blender.org/durian/movies/Sintel.2010.1080p.mkv && \
wget https://download.blender.org/durian/movies/Sintel.2010.4k.mkv

### Big Buck Bunny

wget http://download.blender.org/peach/bigbuckbunny_movies/big_buck_bunny_480p_surround-fix.avi && \
wget http://distribution.bbb3d.renderfarming.net/video/mp4/bbb_sunflower_1080p_60fps_normal.mp4  && \
wget http://distribution.bbb3d.renderfarming.net/video/mp4/bbb_sunflower_2160p_30fps_normal.mp4
