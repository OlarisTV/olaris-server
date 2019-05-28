### 0.1.2 - 28th of Mary 2019

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
