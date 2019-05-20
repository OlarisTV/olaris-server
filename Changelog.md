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
