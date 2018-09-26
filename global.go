// =============================================================================
//  PROJ: pimmp
//  AUTH: ardnew
//  DATE: 26 Sept 2018
//  FILE: global.go
// -----------------------------------------------------------------------------
//
//  DESCRIPTION
//    (TBD)
//
// =============================================================================

package main

import (
	"strings"
)

type MediaKind = byte

const (
	MKUnknown MediaKind = iota
	MKAudio
	MKVideo
)

type FileExtension = map[string][]string

var (
	FEAudio = FileExtension{
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
	}
	FEVideo = FileExtension{
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
	}
)

func fileExtExists(ext string, known FileExtension) (string, bool) {

	for n, e := range known {
		for _, x := range e {
			if x == strings.ToLower(ext) {
				return n, true
			}
		}
	}
	return "", false
}

func FileExtMediaKind(ext string) (MediaKind, string) {

	if n, ok := fileExtExists(ext, FEAudio); ok {
		return MKAudio, n
	}

	if n, ok := fileExtExists(ext, FEVideo); ok {
		return MKVideo, n
	}

	return MKUnknown, ""
}
