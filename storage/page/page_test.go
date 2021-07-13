package page

import (
	"testing"

	testingpkg "github.com/brunocalza/go-bustub/testing"
)

func TestNewPage(t *testing.T) {
	p := New(PageID(0), false, &[PageSize]byte{})

	testingpkg.Equals(t, PageID(0), p.ID())
	testingpkg.Equals(t, uint32(1), p.PinCount())
	p.IncPinCount()
	testingpkg.Equals(t, uint32(2), p.PinCount())
	p.DecPinCount()
	p.DecPinCount()
	p.DecPinCount()
	testingpkg.Equals(t, uint32(0), p.PinCount())
	testingpkg.Equals(t, false, p.IsDirty())
	p.SetIsDirty(true)
	testingpkg.Equals(t, true, p.IsDirty())
	p.CopyToData([]byte{'H', 'E', 'L', 'L', 'O'})
	testingpkg.Equals(t, [PageSize]byte{'H', 'E', 'L', 'L', 'O'}, *p.Data())
}

func TestEmptyPage(t *testing.T) {
	p := NewEmpty(PageID(0))

	testingpkg.Equals(t, PageID(0), p.ID())
	testingpkg.Equals(t, uint32(1), p.PinCount())
	testingpkg.Equals(t, false, p.IsDirty())
	testingpkg.Equals(t, [PageSize]byte{}, *p.Data())
}
