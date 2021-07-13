package disk

import (
	"testing"

	"github.com/brunocalza/go-bustub/storage/page"
	testingpkg "github.com/brunocalza/go-bustub/testing"
)

func TestReadWritePage(t *testing.T) {
	dm := NewDiskManagerTest()
	defer dm.ShutDown()

	data := make([]byte, page.PageSize)
	buffer := make([]byte, page.PageSize)

	copy(data, "A test string.")

	dm.ReadPage(0, buffer) // tolerate empty read
	dm.WritePage(0, data)
	dm.ReadPage(0, buffer)
	testingpkg.Equals(t, data, buffer)

	memset(buffer, 0)
	copy(data, "Another test string.")

	dm.WritePage(5, data)
	dm.ReadPage(5, buffer)
	testingpkg.Equals(t, data, buffer)

	// the size of disk is 24576 bytes because we have 6 pages
	size, err := dm.Size()
	testingpkg.Ok(t, err)
	testingpkg.Equals(t, int64(24576), size)
}

func memset(buffer []byte, value int) {
	for i := range buffer {
		buffer[i] = 0
	}
}
