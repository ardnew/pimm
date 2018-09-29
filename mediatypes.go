// =============================================================================
//  PROJ: pimmp
//  AUTH: ardnew
//  DATE: 26 Sept 2018
//  FILE: mediatypes.go
// -----------------------------------------------------------------------------
//
//  DESCRIPTION
//    defines types related to media and their auxiliary files. also provides
//    subroutines for inspecting and classifying media files.
//
// =============================================================================

package main

import (
	"strings"
)

// enum identifying the different supported types of playable media
type MediaKind int

// concrete values of MediaKind enum type
const (
	mkUnknown MediaKind = iota - 1 // = -1
	mkAudio                        // =  0
	mkVideo                        // =  1
	mkCOUNT                        // =  2
)

// table mapping the name of file types to their common file name extensions
type ExtTable map[string][]string

// struct pairing MediaKind values to their corresponding ExtTable
type MediaExt struct {
	kind  MediaKind
	table *ExtTable
}

var (
	// struct defining how mkAudio media files will be identified through file
	// name inspection. if a file name extension matches at least one string in
	// any of the string slices below, then that file is assumed to be mkAudio.
	// The audio type/encoding of that file is also assumed to be the map key
	// corresponding to the matching string slice.
	AudioExt = MediaExt{
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
	// see discussion of AudioExt above. the same assumptions are made here but
	// with mkVideo and using the VideoExt table below instead.
	VideoExt = MediaExt{
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

func MediaKindOfFileExt(ext string) (MediaKind, string) {

	// iter: all supported kinds of media
	for _, m := range []MediaExt{AudioExt, VideoExt} {
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
