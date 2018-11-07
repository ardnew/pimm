// =============================================================================
//  PROJ: pimmp
//  AUTH: ardnew
//  DATE: 06 Nov 2018
//  FILE: support.go
// -----------------------------------------------------------------------------
//
//  DESCRIPTION
//    defines types related to auxiliary support files and also provides
//    subroutines for inspecting and classifying support files.
//
// =============================================================================

package main

import (
	//"github.com/davecgh/go-spew/spew"

	"encoding/json"
	"os"
	"strings"
)

// type SupportKind is an enum identifying different types of files that support
// media in some way. these files should somehow be associated with the media
// files; they are not necessarily useful on their own.
type SupportKind int

const (
	skUnknown   SupportKind = iota - 1 // = -1
	skSubtitles                        // =  0
	skCOUNT                            // =  1
)

var (
	// variable supportColName maps the SupportKind enum values to the string
	// name of their corresponding collection in the database.
	supportColName = [skCOUNT]string{
		"Subtitles", // 0 = skSubtitles
	}
)

type Support struct {
	*Entity             // common entity info
	Kind    SupportKind // type of support file
}

type Subtitles struct {
	*Support // common support info
}

type SupportIndexID int

const (
	sxPath SupportIndexID = iota
	sxCOUNT
)

var (
	supportIndex = [sxCOUNT]*EntityIndex{
		&EntityIndex{"AbsPath"}, // = sxPath (0)
	}
)

func newSupport(lib *Library, kind SupportKind, absPath, relPath, ext, extName string, info os.FileInfo) *Support {

	entity := newEntity(lib, ecSupport, absPath, relPath, ext, extName, info)

	return &Support{
		Entity: entity, // (*Entity)     common entity info
		Kind:   kind,   // (SupportKind) type of support file
	}
}

func newSubtitles(lib *Library, absPath, relPath, ext, extName string, info os.FileInfo) *Subtitles {

	support := newSupport(lib, skSubtitles, absPath, relPath, ext, extName, info)

	return &Subtitles{
		Support: support, // common support info
	}
}

// type SupportExt is a struct pairing SupportKind values to their corresponding
// ExtTable map.
type SupportExt struct {
	kind  SupportKind
	table *ExtTable
}

var (
	// var subsExt is a struct defining how skSubtitles support files will be
	// identified through file name inspection. if a file name extension matches
	// at least one string in any of the string slices below, then that file is
	// assumed to be skSubtitles. the subtitles type/encoding of that file is
	// also assumed to be the map key corresponding to the matching slice.
	subsExt = SupportExt{
		kind: skSubtitles,
		table: &ExtTable{
			"AQTitle":                    []string{".aqt"},
			"CVD":                        []string{".cvd"},
			"DKS":                        []string{".dks"},
			"Gloss Subtitle":             []string{".gsub"},
			"JACOSub":                    []string{".jss"},
			"MPL2":                       []string{".mpl"},
			"Phoenix Subtitle":           []string{".pjs"},
			"PowerDivX":                  []string{".psb"},
			"RealText / SMIL":            []string{".rt"},
			"SAMI":                       []string{".smi"},
			"SubRip":                     []string{".srt"},
			"SubStation Alpha":           []string{".ssa"},
			"Advanced SubStation Alpha":  []string{".ass"},
			"Structured Subtitle Format": []string{".ssf"},
			"Spruce subtitle format":     []string{".stl"},
			"MicroDVD":                   []string{".sub"},
			"MPSub":                      []string{".sub"},
			"SubViewer":                  []string{".sub"},
			"VobSub":                     []string{".sub", ".idx"},
			"SVCD":                       []string{".svcd"},
			"MPEG-4 Timed Text":          []string{".ttxt"},
			"Universal Subtitle Format":  []string{".usf"},
		},
	}
)

// function supportKindOfFileExt() searches all SupportExt mappings for a given
// file name extension, returning both the SupportKind and the type/encoding
// name associated with that file name extension.
func supportKindOfFileExt(ext string) (SupportKind, string) {

	// constant values in file extension tables are all lowercase. convert the
	// search key to lowercase for case-insensitivity.
	extLower := strings.ToLower(ext)

	// iter: all supported kinds of media
	for _, m := range []SupportExt{subsExt} {
		if n, ok := kindOfFileExt(m.table, extLower); ok {
			return m.kind, n
		}
	}
	return skUnknown, ""
}

// function toRecord() creates a struct capable of being stored in the database.
// defines type Subtitles's implementation of the EntityRecord interface.
func (m *Subtitles) toRecord() (*EntityRecord, *ReturnCode) {

	var (
		record *EntityRecord = &EntityRecord{}
		data   []byte
		err    error
	)

	if data, err = json.Marshal(m); nil != err {
		return nil, rcInvalidJSONData.specf(
			"toRecord(): json.Marshal(%s): cannot marshal Subtitles struct into JSON object: %s", m, err)
	}

	if err = json.Unmarshal(data, record); nil != err {
		return nil, rcInvalidJSONData.specf(
			"toRecord(): json.Unmarshal(%s): cannot unmarshal JSON object into EntityRecord struct: %s", string(data), err)
	}

	return record, nil
}

// function fromRecord() creates a struct using the record stored in the
// database. defines type Subtitles's implementation of the EntityRecord
// interface.
func (m *Subtitles) fromRecord(data []byte) *ReturnCode {

	// Subtitles has an embedded Support struct -pointer- (not struct). so if we
	// create a zeroized Subtitles, the embedded Support will be a null pointer.
	// we can protect this method from that null pointer by creating a zeroized
	// Support and updating Subtitles's embedded pointer to reference it.
	if nil == m.Support {
		m.Support = &Support{}
	}

	// unmarshal our media object directly into the target
	if err := json.Unmarshal(data, m); nil != err {
		return rcInvalidJSONData.specf(
			"fromRecord(): json.Unmarshal(%s): cannot unmarshal JSON object into Subtitles struct: %s", string(data), err)
	}

	return nil
}
