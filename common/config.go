// this code is from https://github.com/pzhzqt/goostub
// there is license and copyright notice in licenses/goostub dir

package common

import (
	"time"
)

var CycleDetectionInterval time.Duration
var EnableLogging bool = false
var LogTimeout time.Duration
var EnableDebug bool = false

const (
	// invalid page id
	InvalidPageID = -1
	// invalid transaction id
	InvalidTxnID = -1
	// invalid log sequence number
	InvalidLSN = -1
	// the header page id
	HeaderPageID = 0
	// size of a data page in byte
	PageSize = 4096
	// size of buffer pool
	BufferPoolSize = 10
	// size of a log buffer in byte
	LogBufferSize = ((BufferPoolSize + 1) * PageSize)
	// size of extendible hash bucket
	BucketSize = 50
)

//type FrameID int32 // frame id type
//type PageID int32       // page id type
type TxnID int32 // transaction id type
//type LSN int32          // log sequence number
type SlotOffset uintptr // slot offset type
//type OID uint16
