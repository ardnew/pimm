# pimmp
Raspberry **PI** **M**edia **M**anager and **P**layer for those who want less

Fast and lightweight ncurses-based interface scans an existing filesystem for content without requiring the files be named or organized in any specific hierarchy. The actual directory structure and supporting files (subtitles, info metadata, etc.) are identified and hidden by the media browser, but they are also silently utilized if available.

It is not necessary to run a graphical window manager for video playback when using Raspbian's handy default video player `omxplayer` (https://github.com/popcornmix/omxplayer) with GPU hardware acceleration, so feel free to save resources and boot directly to command-line. However, the default playback command can be overridden for all media or on a per-media/file basis if you prefer to use mplayer, mpv, VLC, etc. 
