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
	"github.com/HouzuoGuo/tiedot/db"
	//"github.com/davecgh/go-spew/spew"

	"encoding/json"
	"os"
	"path"
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
	*Support        // common support info
	KnownVideoMedia []VideoMedia
}

const (
	// max number of media that can exist in a directory coincidently with a
	// subtitles file to consider them associated (see findCandidates() case 3).
	maxNumMediaAssocSubs int = 2
)

type SupportIndexID int

const (
	sxPath SupportIndexID = iota
	sxDir
	sxName
	sxBase
	sxCOUNT
)

var (
	supportIndex = [sxCOUNT]*EntityIndex{
		&EntityIndex{"AbsPath"}, // = sxPath (0)
		&EntityIndex{"AbsDir"},  // = sxDir  (1)
		&EntityIndex{"AbsName"}, // = sxBase (2)
		&EntityIndex{"AbsBase"}, // = sxBase (3)
	}
)

// function newSupport() creates and initializes a new Support object by
// invoking the embedded types' constructors and then populating any unique
// specialization fields.
func newSupport(lib *Library, kind SupportKind, absPath, relPath, ext, extName string, info os.FileInfo) *Support {

	entity := newEntity(lib, ecSupport, absPath, relPath, ext, extName, info)

	return &Support{
		Entity: entity, // (*Entity)     common entity info
		Kind:   kind,   // (SupportKind) type of support file
	}
}

// function newSubtitles() creates and initializes a new Subtitles object by
// invoking the embedded types' constructors and then populating any unique
// specialization fields.
func newSubtitles(lib *Library, absPath, relPath, ext, extName string, info os.FileInfo) *Subtitles {

	support := newSupport(lib, skSubtitles, absPath, relPath, ext, extName, info)

	return &Subtitles{
		Support:         support, // common support info
		KnownVideoMedia: []VideoMedia{},
	}
}

// function addVideoMedia() adds the given VideoMedia to this Subtitles object
// if and only if the video does not already exist in the object's list of known
// videos. additionally, the database record of these subtitles is also
// optionally updated to store the video in the list of known VideoMedia.
func (s *Subtitles) addVideoMedia(col *db.Col, id int, update bool, vid *VideoMedia) (bool, *ReturnCode) {

	var (
		rec     *EntityRecord
		recErr  *ReturnCode
		vidSeen bool
	)

	// walk the current list of known videos, setting a flag if we have already
	// seen this one before.
	for _, v := range s.KnownVideoMedia {
		if v.AbsPath == vid.AbsPath {
			vidSeen = true
			break
		}
	}
	// append it to the list if we haven't seen it before.
	if !vidSeen {
		s.KnownVideoMedia = append(s.KnownVideoMedia, *vid)
	}

	// update the database record of this Subtitles to include the new
	// video association. any subsequent queries should thus include it.
	if rec, recErr = s.toRecord(); nil != recErr {
		return false, recErr
	}
	if update {
		if err := col.Update(id, *rec); nil != err {
			return false, rcDatabaseError.specf(
				"addVideoMedia(%s, %d, %s): failed to update record: %s", col, id, *vid, err)
		}
	}

	// return true if and only if we added this subtitles reference to the list.
	// return(ed) false if we've either seen it before or if there was an error
	// somewhere (e.g. updating the database).
	return !vidSeen, nil
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

// function isInSubtitlesSubdir() inspects this subtitles file's absolute file
// path to determine if one of its parent directories is one of the known,
// common names typically used to store subtitles in a directory relative to the
// location of a video media file. if found, an absolute path to the parent of
// the deepest matching directory found is returned.
func (s *Subtitles) isInSubtitlesSubdir() (bool, string) {

	dir := strings.Split(s.AbsDir, pathSep)
	for i := len(dir) - 1; i >= 0; i-- {
		switch name := dir[i]; strings.ToLower(name) {
		case "sub", "subs", "subtitle", "subtitles", "vobsub", "srt":
			if i > 1 {
				return true, strings.Join(dir[:i], pathSep)
			} else {
				if i > 0 {
					return true, pathSep
				} else {
					return true, currDir
				}
			}
		}
	}
	return false, ""
}

// function toRecord() creates a struct capable of being stored in the database.
// defines type Subtitles's implementation of the StorableEntity interface.
func (s *Subtitles) toRecord() (*EntityRecord, *ReturnCode) {

	var (
		record *EntityRecord = &EntityRecord{}
		data   []byte
		err    error
	)

	if data, err = json.Marshal(s); nil != err {
		return nil, rcInvalidJSONData.specf(
			"toRecord(): json.Marshal(%s): cannot marshal Subtitles struct into JSON object: %s", s, err)
	}

	if err = json.Unmarshal(data, record); nil != err {
		return nil, rcInvalidJSONData.specf(
			"toRecord(): json.Unmarshal(%s): cannot unmarshal JSON object into EntityRecord struct: %s", string(data), err)
	}

	return record, nil
}

// function fromRecord() creates a struct using the record stored in the
// database. defines type Subtitles's implementation of the StorableEntity
// interface.
func (s *Subtitles) fromRecord(data []byte) *ReturnCode {

	// Subtitles has an embedded Support struct -pointer- (not struct). so if we
	// create a zeroized Subtitles, the embedded Support will be a null pointer.
	// we can protect this method from that null pointer by creating a zeroized
	// Support and updating Subtitles's embedded pointer to reference it.
	if nil == s.Support {
		s.Support = &Support{}
	}

	// unmarshal our media object directly into the target
	if err := json.Unmarshal(data, s); nil != err {
		return rcInvalidJSONData.specf(
			"fromRecord(): json.Unmarshal(%s): cannot unmarshal JSON object into Subtitles struct: %s", string(data), err)
	}

	return nil
}

// function fromID() creates a concrete Subtitles struct using the record
// stored in the given collection with the given hash key id.
func (s *Subtitles) fromID(col *db.Col, id int) *ReturnCode {

	read, readErr := col.Read(id)
	if nil != readErr {
		return rcDatabaseError.specf(
			"fromID(%s): db.Read(%d): cannot read record from database: %s",
			col, id, readErr)
	}

	data, marshalErr := json.Marshal(read)
	if nil != marshalErr {
		return rcInvalidJSONData.specf(
			"fromID(%s): json.Marshal(%s): cannot marshal query result into JSON object: %s",
			col, read, marshalErr)
	}

	unmarshalErr := json.Unmarshal(data, s)
	if nil != unmarshalErr {
		return rcInvalidJSONData.specf(
			"fromID(%s): json.Unmarshal(%s): cannot unmarshal JSON object into Subtitles struct: %s",
			col, data, unmarshalErr)
	}

	return nil
}

// function findCandidates() scans the database for video media that appears to
// be related to this subtitles file in some nominal/positional way. [NOTE that
// the string evaluations are currently all case-sensitive comparisons. this is
// a limitation of the database engine being used. if it becomes a significant
// problem, we can create additional fields in the records that have a fixed,
// known character case and use those fields for the queries.]
// if argument update is true, then the database is updated to store all of the
// bi-directional associations discovered between this Subtitles object and its
// VideoMedia objects. the argument subID is the current doc ID of this
// Subtitles object in the given library's subtitles table (only required if
// update is true).
func (s *Subtitles) findCandidates(lib *Library, update bool, subID int) ([]*VideoMedia, *ReturnCode) {

	var (
		queryResult map[int]struct{}
		query       []interface{}
		addErr      *ReturnCode
		added       bool
	)

	vidCol := lib.db.col[ecMedia][mkVideo]
	subCol := lib.db.col[ecSupport][skSubtitles]
	idx := lib.db.index[ecMedia]
	candidate := []*VideoMedia{}

	queryResult = make(map[int]struct{})
	query = []interface{}{
		// first check: does the base name of the subtitles file match exactly with
		// the base name of any media file?
		//   e.g., "Foo.avi" <- "Foo.srt"
		map[string]interface{}{
			"eq": s.AbsBase,
			"in": []interface{}{(*idx[mxBase])[0]},
		},
		// second: does the subtitles file exist in a directory whose name matches
		// exactly with the base name of any media file?
		//   e.g., "/a/b/Foo/Foo.avi" <- "/a/b/Foo/Bar.srt"
		map[string]interface{}{
			"n": []interface{}{
				map[string]interface{}{
					"eq": s.AbsDir,
					"in": []interface{}{(*idx[mxDir])[0]},
				},
				map[string]interface{}{
					"eq": path.Base(s.AbsDir),
					"in": []interface{}{(*idx[mxBase])[0]},
				},
			},
		},
	}

	// third: do the subtitles exist in a directory with a common name for
	// subtitles dirs and that subdir exists in the same dir as a media file?
	//   e.g., "/a/b/Foo.avi" <- "/a/b/Subs/Bar.srt"
	if found, dir := s.isInSubtitlesSubdir(); found {
		query = append(query,
			map[string]interface{}{
				"eq": dir,
				"in": []interface{}{(*idx[mxDir])[0]},
			})
	}

	if err := db.EvalQuery(query, vidCol, &queryResult); nil != err {
		return nil, rcQueryError.specf(
			"findCandidates(%s): EvalQuery({%s, %s}): %s", lib, s.AbsBase, *idx[mxBase], err)
	}
	for id := range queryResult {
		video := &VideoMedia{}
		video.fromID(vidCol, id)
		if added, addErr = video.addSubtitles(vidCol, subCol, id, subID, update, false, s); nil != addErr {
			return nil, addErr
		}
		if added {
			infoLog.tracef("associated subtitles (%q, [type-a]) with video: %q",
				s.AbsName, video.Name)
			candidate = append(candidate, video)
		}
	}

	// only continue on with an additional query if we still haven't found any
	// candidates yet. otherwise, trust that one of the other methods have a far
	// more likely candidate.
	if 0 == len(candidate) {
		// fourth: does the subtitles file exist in a directory that has <= N media
		// files? using N is just a heuristic -- it prevents a subtitles file
		// being selected for every video in a directory containing a large number
		// of videos, but it also allows for subtitles to be selected when they
		// exist in a directory containing very few media files yet don't have a
		// consistent or similar base file name.
		//   e.g. (N=2), {"Foo1.avi","Foo2.avi"} <- "Bar.srt"
		queryResult = make(map[int]struct{})
		query = []interface{}{
			map[string]interface{}{
				"eq": s.AbsDir,
				"in": []interface{}{(*idx[mxDir])[0]},
			},
		}
		if err := db.EvalQuery(query, vidCol, &queryResult); nil != err {
			return nil, rcQueryError.specf(
				"findCandidates(%s): EvalQuery({%s, %s}): %s", lib, s.AbsBase, *idx[mxBase], err)
		}
		if len(queryResult) <= maxNumMediaAssocSubs {
			for id := range queryResult {
				video := &VideoMedia{}
				video.fromID(vidCol, id)
				if added, addErr = video.addSubtitles(vidCol, subCol, id, subID, update, false, s); nil != addErr {
					return nil, addErr
				}
				if added {
					infoLog.tracef("associated subtitles (%q, [type-b]) with video: %q",
						s.AbsName, video.Name)
					candidate = append(candidate, video)
				}
			}
		}
	}

	return candidate, nil
}
