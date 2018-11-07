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
	//"github.com/davecgh/go-spew/spew"

	"encoding/json"
	"os"
	"strings"
	"time"
)

// type MediaKind is an enum identifying the different supported types of
// playable media.
type MediaKind int

const (
	mkUnknown MediaKind = iota - 1 // = -1
	mkAudio                        // =  0
	mkVideo                        // =  1
	mkCOUNT                        // =  2
)

// variable mediaColName maps the MediaKind enum values to the string name of
// their corresponding collection in the database.
var (
	mediaColName = [mkCOUNT]string{
		"Audio", // 0 = mkAudio
		"Video", // 1 = mkVideo
	}
)

// type Media is used to reference every kind of playable media -- the struct
// fields are common among both audio and video.
type Media struct {
	// fixed, read-only system info
	*Entity           // common entity info
	Kind    MediaKind // type of media
	// user-writable system info
	Name            string    // displayed name
	TimeAdded       time.Time // date media was discovered and added to library
	PlaybackCommand string    // full system command used to play media
	// user-writable public media info
	Title       string    // official name of media
	Description string    // synopsis/summary of media content
	ReleaseDate time.Time // date media was produced/released
}

// type AudioMedia is a specialized type of media containing struct fields
// relevant only to video.
type AudioMedia struct {
	*Media        // common media info
	Album  string // name of the album on which the track appears
	Track  int64  // numbered index of where track is located on album
}

// type VideoMedia is a specialized type of media containing struct fields
// relevant only to audio.
type VideoMedia struct {
	*Media                     // common media info
	KnownSubtitles []Subtitles // absolute path to all associated subtitles
	Subtitles      Subtitles   // absolute path to selected subtitles
}

type MediaIndexID int

const (
	mxPath MediaIndexID = iota
	mxCOUNT
)

var (
	mediaIndex = [mxCOUNT]*EntityIndex{
		&EntityIndex{"AbsPath"}, // = mxPath (0)
	}
)

// function newMedia() creates and initializes a new Media object. the kind
// of media is determined automatically, and the MediaKind field is set
// accordingly. so once the media has been identified, a type assertion can be
// performed to handle the object appropriately and unambiguously.
func newMedia(lib *Library, kind MediaKind, absPath, relPath, ext, extName string, info os.FileInfo) *Media {

	entity := newEntity(lib, ecMedia, absPath, relPath, ext, extName, info)

	return &Media{
		Entity:          entity,      // (*Entity)   common entity info
		Kind:            kind,        // (MediaKind) type of media
		Name:            info.Name(), // (string)    displayed name
		TimeAdded:       time.Now(),  // (time.Time) date media was discovered and added to library
		PlaybackCommand: "--",        // (string)    full system command used to play media
		Title:           info.Name(), // (string)    official name of media
		Description:     "--",        // (string)    synopsis/summary of media content
		ReleaseDate:     time.Time{}, // (time.Time) date media was produced/released
	}
}

func newAudioMedia(lib *Library, absPath, relPath, ext, extName string, info os.FileInfo) *AudioMedia {

	media := newMedia(lib, mkAudio, absPath, relPath, ext, extName, info)

	return &AudioMedia{
		Media: media, // common media info
		Album: "",    // name of the album on which the track appears
		Track: -1,    // numbered index of where track is located on album
	}
}

func newVideoMedia(lib *Library, absPath, relPath, ext, extName string, info os.FileInfo) *VideoMedia {

	media := newMedia(lib, mkVideo, absPath, relPath, ext, extName, info)

	return &VideoMedia{
		Media:          media,         // common media info
		KnownSubtitles: []Subtitles{}, // absolute path to all associated subtitles
		Subtitles:      Subtitles{},   // absolute path to selected subtitles
	}
}

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
			"3D Solar UK Ltd":               []string{".ivs"},
			"ACT Lossy ADPCM":               []string{".act"},
			"Adaptive Multi-Rate":           []string{".amr"},
			"Adaptive Multi-Rate Wideband":  []string{".awb"},
			"Advanced Audio Coding":         []string{".aac"},
			"Apple AIFF":                    []string{".aiff"},
			"Audible Audiobook":             []string{".aa", ".aax"},
			"Dialogic ADPCM":                []string{".vox"},
			"Digital Speech Standard":       []string{".dss"},
			"Electronic Arts IFF-8SVX":      []string{".8svx"},
			"Free Lossless Audio Codec":     []string{".flac"},
			"GSM Telephony":                 []string{".gsm"},
			"iKlax Media":                   []string{".iklax"},
			"Linear PCM":                    []string{".sln"},
			"Microsoft WAV":                 []string{".wav"},
			"Microsoft Windows Media Audio": []string{".wma"},
			"Monkey's Audio":                []string{".ape"},
			"MPEG Layer III":                []string{".mp3"},
			"MPEG-4 Part 14":                []string{".m4a", ".m4b"},
			"Musepack/MPC/MPEG":             []string{".mpc"},
			"NCH Dictation":                 []string{".dct"},
			"Nintendo (NES) Sound Format":   []string{".nsf"},
			"Ogg Audio":                     []string{".oga", ".mogg"},
			"Opus":                          []string{".opus"},
			"RAW Audio Format":              []string{".raw"},
			"RealAudio":                     []string{".ra"},
			"Samsung Yamaha Ringtone":       []string{".mmf"},
			"Sony Compressed Voice":         []string{".dvf", ".msv"},
			"Sun/Unix/Java Audio":           []string{".au"},
			"True Audio Lossless":           []string{".tta"},
			"WavPack":                       []string{".wv"},
		},
	}
	// var videoExt is a struct defining how mkAudio media files will be
	// identified through file name inspection. see discussion of audioExt
	// above. the same assumptions are made here but with mkVideo instead.
	videoExt = MediaExt{
		kind: mkVideo,
		table: &ExtTable{
			"3GPP":                              []string{".3gp"},
			"3GPP2":                             []string{".3g2"},
			"Advanced Systems Format":           []string{".asf"},
			"AMV video format":                  []string{".amv"},
			"Audio Video Interleave":            []string{".avi"},
			"Dirac":                             []string{".drc"},
			"Flash Video":                       []string{".flv", ".f4v", ".f4p", ".f4a", ".f4b"},
			"Graphics Interchange Format Video": []string{".gifv"},
			"Material Exchange Format":          []string{".mxf"},
			"Matroska":                          []string{".mkv"},
			"MPEG Transport Stream":             []string{".mts", ".m2ts"},
			"MPEG-1":                            []string{".mp2", ".mpe", ".mpv"},
			"MPEG-1/MPEG-2":                     []string{".mpg", ".mpeg"},
			"MPEG-2":                            []string{".m2v"},
			"MPEG-4 Part 14":                    []string{".mp4", ".m4p", ".m4v"},
			"Multiple-image Network Graphics":   []string{".mng"},
			"Nullsoft Streaming Video":          []string{".nsv"},
			"Ogg Video":                         []string{".ogv", ".ogg"},
			"QuickTime File Format":             []string{".mov", ".qt"},
			"Raw video format":                  []string{".yuv"},
			"RealMedia":                         []string{".rm"},
			"RealMedia Variable Bitrate":        []string{".rmvb"},
			"RoQ FMV":                           []string{".roq"},
			"Standardized Video Interview":      []string{".svi"},
			"Video Object":                      []string{".vob"},
			"WebM":                              []string{".webm"},
			"Windows Media Video":               []string{".wmv"},
		},
	}
)

// function mediaKindOfFileExt() searches all MediaExt mappings for a given
// file name extension, returning both the MediaKind and the type/encoding name
// associated with that file name extension.
func mediaKindOfFileExt(ext string) (MediaKind, string) {

	// constant values in file extension tables are all lowercase. convert the
	// search key to lowercase for case-insensitivity.
	extLower := strings.ToLower(ext)

	// iter: all supported kinds of media
	for _, m := range []MediaExt{audioExt, videoExt} {
		if n, ok := kindOfFileExt(m.table, extLower); ok {
			return m.kind, n
		}
	}
	return mkUnknown, ""
}

// function toRecord() creates a struct capable of being stored in the database.
// defines type AudioMedia's implementation of the EntityRecord interface.
func (m *AudioMedia) toRecord() (*EntityRecord, *ReturnCode) {

	var (
		record *EntityRecord = &EntityRecord{}
		data   []byte
		err    error
	)

	if data, err = json.Marshal(m); nil != err {
		return nil, rcInvalidJSONData.specf(
			"toRecord(): json.Marshal(%s): cannot marshal AudioMedia struct into JSON object: %s", m, err)
	}

	if err = json.Unmarshal(data, record); nil != err {
		return nil, rcInvalidJSONData.specf(
			"toRecord(): json.Unmarshal(%s): cannot unmarshal JSON object into EntityRecord struct: %s", string(data), err)
	}

	return record, nil
}

// function fromRecord() creates a struct using the record stored in the
// database. defines type AudioMedia's implementation of the EntityRecord
// interface.
func (m *AudioMedia) fromRecord(data []byte) *ReturnCode {

	// AudioMedia has an embedded Media struct -pointer- (not struct). so if we
	// create a zeroized AudioMedia, the embedded Media will be a null pointer.
	// we can protect this method from that null pointer by creating a zeroized
	// Media and updating AudioMedia's embedded pointer to reference it.
	if nil == m.Media {
		m.Media = &Media{}
	}

	// unmarshal our media object directly into the target
	if err := json.Unmarshal(data, m); nil != err {
		return rcInvalidJSONData.specf(
			"fromRecord(): json.Unmarshal(%s): cannot unmarshal JSON object into AudioMedia struct: %s", string(data), err)
	}

	return nil
}

// function toRecord() creates a struct capable of being stored in the database.
// defines type VideoMedia's implementation of the EntityRecord interface.
func (m *VideoMedia) toRecord() (*EntityRecord, *ReturnCode) {

	var (
		record *EntityRecord = &EntityRecord{}
		data   []byte
		err    error
	)

	if data, err = json.Marshal(m); nil != err {
		return nil, rcInvalidJSONData.specf(
			"toRecord(): json.Marshal(%s): cannot marshal VideoMedia struct into JSON object: %s", m, err)
	}

	if err = json.Unmarshal(data, record); nil != err {
		return nil, rcInvalidJSONData.specf(
			"toRecord(): json.Unmarshal(%s): cannot unmarshal JSON object into EntityRecord struct: %s", string(data), err)
	}

	return record, nil
}

// function fromRecord() creates a struct using the record stored in the
// database. defines type VideoMedia's implementation of the EntityRecord
// interface.
func (m *VideoMedia) fromRecord(data []byte) *ReturnCode {

	// VideoMedia has an embedded Media struct -pointer- (not struct). so if we
	// create a zeroized VideoMedia, the embedded Media will be a null pointer.
	// we can protect this method from that null pointer by creating a zeroized
	// Media and updating VideoMedia's embedded pointer to reference it.
	if nil == m.Media {
		m.Media = &Media{}
	}

	// unmarshal our media object directly into the target
	if err := json.Unmarshal(data, m); nil != err {
		return rcInvalidJSONData.specf(
			"fromRecord(): json.Unmarshal(%s): cannot unmarshal JSON object into VideoMedia struct: %s", string(data), err)
	}

	return nil
}
