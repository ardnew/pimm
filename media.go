// =============================================================================
//  PROJ: pimmp
//  AUTH: ardnew
//  DATE: 26 Sept 2018
//  FILE: media.go
// -----------------------------------------------------------------------------
//
//  DESCRIPTION
//    (TBD)
//
// =============================================================================

package main

type Media struct {
	path            string    // absolute path to media file
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
	knownSubtitles []string // absolute path to all subtitles associated
	subtitles      string   // absolute path to selected subtitles
}

type MediaAudio struct {
	*Media
}
