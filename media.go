package main

type MediaKind = byte

const (
	MKAudio MediaKind = 0
	MKVideo MediaKind = 1
)

type Media struct {
	path            string    // full path to media file
	kind            MediaKind // type of media
	name            string    // displayed name
	dateAdded       string    // date media was discovered and added to library
	playbackCommand string    // full system command used to play media

	title       string // official name of media
	description string // synopsis/summary of media content
	releaseDate string // date media was produced/released
}

type MediaVideo struct {
	*Media
	knownSubtitles []string // full path to all subtitles associated
	subtitles      string   // full path to selected subtitles
}

type MediaAudio struct {
	*Media
}
