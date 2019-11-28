### 0.3.0 - 2019-11-18

* Chromecast support
* Retagging mistagged TV Shows and Movies as well as untagged movies
* Improved user administration page
* Many style fixes and smaller visual improvements here and there
* Fixed lots of little race conditions, refactorings here and there

### 0.2.0 - 21th of June 2019

* Multiple library backend support
* Rclone backend support
* Pagination support with auto-loading in the React app
* Increased filename to content name compatitiblity
* Fixed race condition in ffprobe
* Better movie merging logic

### 0.1.2 - 28th of May 2019

* Faster library scanning, should be up to four times faster in this
  release.
* Libraries will now show a spinner when they are scanning.
* Improved watching for file changes, files added to previously empty
  folders should now also trigger a library rescan.

* Fixed an issue where the next episode was not properly shown in upNext
  when you finished a season.
* Fixed an issue where adding libraries too fast would kill the scanning
  of the first added.
* Long folder names were being cut-off in the "Add library" screen.
* Fixed a race condition that could add series multiple times.


### 0.1.1 - 20th of May 2019

* Added a default basepath, this makes it possible to proxy via a third
  party webserver.
* Added favicon.
* Some basic Chromecast support should now load as long as you access
  the React application over SSL.

* Fixed Playstate missing from Dashboard
* Fixed ffmpeg feedback port
* Fixed new user validation feedback

* Changed Sqlite connection to use WAL.
* Changed dumplogs to dump logs without colons in them
* Changed user creation via CLI to not load the whole metadata
  environment
