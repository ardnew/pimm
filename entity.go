// =============================================================================
//  PROJ: pimmp
//  AUTH: ardnew
//  DATE: 06 Nov 2018
//  FILE: entity.go
// -----------------------------------------------------------------------------
//
//  DESCRIPTION
//    defines types related to file entities, which is the base type of all real
//    files stored on disk (media files, subtitles, etc.).
//
// =============================================================================

package main

import (
	"github.com/HouzuoGuo/tiedot/db"
	//"github.com/davecgh/go-spew/spew"

	"fmt"
	"os"
	"path"
	"strings"
	"time"
)

// type EntityClass is an enum identifying the different types of file entities
// stored in the persistent database.
type EntityClass int

const (
	ecUnknown EntityClass = iota - 1 // = -1
	ecMedia                          // =  0
	ecSupport                        // =  1
	ecCOUNT                          // =  2
)

var (
	entityColName = [ecCOUNT][]string{
		mediaColName[:],   // 0 = ecMedia
		supportColName[:], // 1 = ecSupport
	}
	entityIndex = [ecCOUNT][]*EntityIndex{
		mediaIndex[:],   // 0 = ecMedia
		supportIndex[:], // 1 = ecSupport
	}
)

// type Entity is used to describe any sort of file encountered on the file
// system. this includes not just audio/video media, but also subtitles and any
// other auxiliary data files.
type Entity struct {
	Class        EntityClass // type of entity
	AbsPath      string      // absolute path to media file
	AbsDir       string      // directory portion of AbsPath
	AbsName      string      // file name portion of AbsPath
	AbsBase      string      // AbsName without file name extension
	RelPath      string      // CWD-relative path to media file
	Size         int64       // length in bytes for regular files; system-dependent for others
	Mode         os.FileMode // file mode bits
	TimeModified time.Time   // modification time
	SysInfo      interface{} // underlying data source (can return nil)
	Ext          string      // file name extension
	ExtName      string      // name of file type/encoding (per file name extension)
}

// type EntityRecord represents the struct stored in the database for an
// individual media or support record
type EntityRecord map[string]interface{}
type EntityIndex []string
type StorableEntity interface {
	toRecord() (*EntityRecord, *ReturnCode)
	fromRecord([]byte) *ReturnCode
	fromID(*db.Col, int) *ReturnCode
}

func newEntity(lib *Library, class EntityClass, absPath, relPath, ext, extName string, info os.FileInfo) *Entity {

	// the lack of file name extension abstracts any encoding info from the
	// release name of the media, convenient for lookup via indexed queries.
	absBase := strings.TrimSuffix(info.Name(), ext)

	return &Entity{
		Class:        class,             // (EntityClass) type of entity
		AbsPath:      absPath,           // (string)      absolute path to media file
		AbsDir:       path.Dir(absPath), // (string)      directory portion of AbsPath
		AbsName:      info.Name(),       // (string)      file name portion of AbsPath
		AbsBase:      absBase,           // (string)      AbsName without file name extension
		RelPath:      relPath,           // (string)      CWD-relative path to media file
		Size:         info.Size(),       // (int64)       length in bytes for regular files; system-dependent for others
		Mode:         info.Mode(),       // (os.FileMode) file mode bits
		TimeModified: info.ModTime(),    // (time.Time)   modification time
		SysInfo:      info.Sys(),        // (interface{}) underlying data source (can return nil)
		Ext:          ext,               // (string)      file name extension
		ExtName:      extName,           // (string)      name of file type/encoding (per file name extension)
	}
}

func (e *Entity) String() string {
	path := e.AbsPath
	if "" != e.RelPath && len(e.RelPath) < len(e.AbsPath) {
		path = e.RelPath
	}
	return fmt.Sprintf("%s [%s (%s)] (%d bytes) %v",
		path, e.ExtName, e.Ext, e.Size, e.TimeModified)
}
