// =============================================================================
//  PROJ: pimmp
//  AUTH: ardnew
//  DATE: 26 Sept 2018
//  FILE: media.go
// -----------------------------------------------------------------------------
//
//  DESCRIPTION
//    defines types related to media and their auxiliary files and also provides
//    subroutines for inspecting and classifying media files.
//
// =============================================================================

package main

import (
	"fmt"
	"os"
	"strings"
)

// type Media is used to reference every kind of playable media -- the struct
// fields are common among both audio and video.
type Media struct {
	absPath         string      // absolute path to media file
	relPath         string      // CWD-relative path to media file
	info            os.FileInfo // info stored on the file system
	kind            MediaKind   // type of media
	name            string      // displayed name
	dateAdded       string      // date media was discovered and added to library
	playbackCommand string      // full system command used to play media

	title       string // official name of media
	description string // synopsis/summary of media content
	releaseDate string // date media was produced/released
}

// type MediaAudio is a specialized type of media containing struct fields
// relevant only to video.
type MediaAudio struct {
	*Media
	album string // name of the album on which the track appears
	track int    // numbered index of where track is located on album
}

// type MediaVideo is a specialized type of media containing struct fields
// relevant only to audio.
type MediaVideo struct {
	*Media
	knownSubtitles []string // absolute path to all associated subtitles
	subtitles      string   // absolute path to selected subtitles
}

// type MediaKind is an enum identifying the different supported types of
// playable media.
type MediaKind int

// function newMedia() creates and initialized a new Media object. the kind
// of media is determined automatically, and the MediaKind field is set
// accordingly. so once the media has been identified, a type assertion can be
// performed to handle the object appropriately and unambiguously.
func newMedia(lib *Library, absPath, relPath string, info os.FileInfo) (*Media, *ReturnCode) {
	return &Media{absPath, relPath, info, mkUnknown, info.Name(), "", "", "", "", ""}, nil
}

// creates a string representation of the Media for easy identification in logs.
func (m *Media) String() string {
	path := m.absPath
	if "" != m.relPath {
		path = m.relPath
	}
	return fmt.Sprintf("%s (%d bytes) %v", path, m.info.Size(), m.info.ModTime())
}

// concrete values of the MediaKind enum type.
const (
	mkUnknown MediaKind = iota - 1 // = -1
	mkAudio                        // =  0
	mkVideo                        // =  1
	mkCOUNT                        // =  2
)

// type ExtTable is a mapping of the name of file types to their common file
// name extensions.
type ExtTable map[string][]string

// type MediaExt is a struct pairing MediaKind values to their corresponding
// ExtTable map.
type MediaExt struct {
	kind  MediaKind
	table *ExtTable
}

var (
	// var audioExt is a struct defining how mkAudio media files will be
	// identified through file name inspection. if a file name extension matches
	// at least one string in any of the string slices below, then that file is
	// assumed to be mkAudio. the audio type/encoding of that file is also
	// assumed to be the map key corresponding to the matching string slice.
	audioExt = MediaExt{
		kind: mkAudio,
		table: &ExtTable{
			"3D Solar UK Ltd":               []string{"ivs"},
			"ACT Lossy ADPCM":               []string{"act"},
			"Adaptive Multi-Rate":           []string{"amr"},
			"Adaptive Multi-Rate Wideband":  []string{"awb"},
			"Advanced Audio Coding":         []string{"aac"},
			"Apple AIFF":                    []string{"aiff"},
			"Audible Audiobook":             []string{"aa", "aax"},
			"Dialogic ADPCM":                []string{"vox"},
			"Digital Speech Standard":       []string{"dss"},
			"Electronic Arts IFF-8SVX":      []string{"8svx"},
			"Free Lossless Audio Codec":     []string{"flac"},
			"GSM Telephony":                 []string{"gsm"},
			"iKlax Media":                   []string{"iklax"},
			"Linear PCM":                    []string{"sln"},
			"Microsoft WAV":                 []string{"wav"},
			"Microsoft Windows Media Audio": []string{"wma"},
			"Monkey's Audio":                []string{"ape"},
			"MPEG Layer III":                []string{"mp3"},
			"MPEG-4 Part 14":                []string{"m4a", "m4b"},
			"Musepack/MPC/MPEG":             []string{"mpc"},
			"NCH Dictation":                 []string{"dct"},
			"Nintendo (NES) Sound Format":   []string{"nsf"},
			"Ogg Audio":                     []string{"oga", "mogg"},
			"Opus":                          []string{"opus"},
			"RAW Audio Format":              []string{"raw"},
			"RealAudio":                     []string{"ra"},
			"Samsung Yamaha Ringtone":       []string{"mmf"},
			"Sony Compressed Voice":         []string{"dvf", "msv"},
			"Sun/Unix/Java Audio":           []string{"au"},
			"True Audio Lossless":           []string{"tta"},
			"WavPack":                       []string{"wv"},
		},
	}
	// var videoExt is a struct defining how mkAudio media files will be
	// identified through file name inspection. see discussion of audioExt
	// above. the same assumptions are made here but with mkVideo instead.
	videoExt = MediaExt{
		kind: mkVideo,
		table: &ExtTable{
			"3GPP":                              []string{"3gp"},
			"3GPP2":                             []string{"3g2"},
			"Advanced Systems Format":           []string{"asf"},
			"AMV video format":                  []string{"amv"},
			"Audio Video Interleave":            []string{"avi"},
			"Dirac":                             []string{"drc"},
			"Flash Video":                       []string{"flv", "f4v", "f4p", "f4a", "f4b"},
			"Graphics Interchange Format Video": []string{"gifv"},
			"Material Exchange Format":          []string{"mxf"},
			"Matroska":                          []string{"mkv"},
			"MPEG Transport Stream":             []string{"mts", "m2ts"},
			"MPEG-1":                            []string{"mp2", "mpe", "mpv"},
			"MPEG-1/MPEG-2":                     []string{"mpg", "mpeg"},
			"MPEG-2":                            []string{"m2v"},
			"MPEG-4 Part 14":                    []string{"mp4", "m4p", "m4v"},
			"Multiple-image Network Graphics":   []string{"mng"},
			"Nullsoft Streaming Video":          []string{"nsv"},
			"Ogg Video":                         []string{"ogv", "ogg"},
			"QuickTime File Format":             []string{"mov", "qt"},
			"Raw video format":                  []string{"yuv"},
			"RealMedia":                         []string{"rm"},
			"RealMedia Variable Bitrate":        []string{"rmvb"},
			"RoQ FMV":                           []string{"roq"},
			"Standardized Video Interview":      []string{"svi"},
			"Video Object":                      []string{"vob"},
			"WebM":                              []string{"webm"},
			"Windows Media Video":               []string{"wmv"},
		},
	}
)

// function mediaKindOfFileExt() searches all MediaExt mappings for a given
// file name extension, returning both the MediaKind and the type/encoding name
// associated with that file name extension.
func mediaKindOfFileExt(ext string) (MediaKind, string) {

	// iter: all supported kinds of media
	for _, m := range []MediaExt{audioExt, videoExt} {
		// iter: each entry in current media's file extension table
		for n, l := range *(m.table) {
			// iter: each file extension in current table entry
			for _, e := range l {
				// cond: wanted file extension matches current file extension
				if e == strings.ToLower(ext) {
					// return: current media kind, file type of extension
					return m.kind, n
				}
			}
		}
	}
	return mkUnknown, ""
}
