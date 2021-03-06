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
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/HouzuoGuo/tiedot/db"
	//"github.com/davecgh/go-spew/spew"
)

// type EntityClass is an enum identifying the different types of file entities
// stored in the persistent database.
type EntityClass int

// constant enum IDs for the various structs that embed/subclass Entity.
const (
	ecUnknown EntityClass = iota - 1 // = -1
	ecMedia                          // =  0
	ecSupport                        // =  1
	ecCOUNT                          // =  2
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

// type EntityIndex is a slice in which each string element corresponds to the
// name of the corresponding type's struct field used as a database column that
// needs to be indexed for searching purposes.
type EntityIndex []string

// type StorableEntity defines the functions that must be defined for any struct
// that embeds/subclasses Entity and supports storage in the database engine.
// note that Entity itself does not implement these functions!
type StorableEntity interface {
	toRecord() (*EntityRecord, *ReturnCode)
	fromRecord([]byte) *ReturnCode
	fromID(*db.Col, int) *ReturnCode
}

// storage for the names and database indices for each enum ID of the various
// structs that embed/subclass Entity.
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

// function newEntity() creates a new file object that serves as the fundamental
// type constituting any sort of file capable of being referenced on the file
// system. this includes media files, supporting auxiliary files, etc.
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

// function String() creates a string representation of the Entity for easy
// identification in logs.
func (e *Entity) String() string {
	path := e.AbsPath
	if "" != e.RelPath && len(e.RelPath) < len(e.AbsPath) {
		path = e.RelPath
	}
	return fmt.Sprintf("\"%s\" [%s (%s)] (%d bytes) %v",
		path, e.ExtName, e.Ext, e.Size, e.TimeModified)
}
