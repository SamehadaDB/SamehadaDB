// this code is from https://github.com/brunocalza/go-bustub
// there is license and copyright notice in licenses/go-bustub dir

package catalog

import (
	"fmt"
	"os"
	"testing"

	"github.com/ryogrid/SamehadaDB/storage/buffer"
	"github.com/ryogrid/SamehadaDB/storage/table/column"
	"github.com/ryogrid/SamehadaDB/storage/table/schema"
	"github.com/ryogrid/SamehadaDB/test_util"
	testingpkg "github.com/ryogrid/SamehadaDB/testing"
	"github.com/ryogrid/SamehadaDB/types"
)

// test reloading serialized catalog info in db file at lauching system
func TestTableCatalogReload(t *testing.T) {
	os.Remove("test.db")
	samehada_instance := test_util.NewSamehadaInstance()
	//diskManager := disk.NewDiskManagerImpl("test.db")
	//defer diskManager.ShutDown()
	bpm := buffer.NewBufferPoolManager(uint32(32), samehada_instance.GetDiskManager(), samehada_instance.GetLogManager())

	txn := samehada_instance.GetTransactionManager().Begin(nil)
	catalog_old := BootstrapCatalog(bpm, samehada_instance.GetLogManager(), samehada_instance.GetLockManager(), txn)

	columnA := column.NewColumn("a", types.Integer, false, nil)
	columnB := column.NewColumn("b", types.Integer, true, nil)
	schema_ := schema.NewSchema([]*column.Column{columnA, columnB})

	catalog_old.CreateTable("test_1", schema_, txn)
	bpm.FlushAllPages()

	fmt.Println("Shutdown system...")

	samehada_instance_new := test_util.NewSamehadaInstance()
	txn_new := samehada_instance_new.GetTransactionManager().Begin(nil)
	//catalog := GetCatalog(bpm)
	catalog_recov := RecoveryCatalogFromCatalogPage(samehada_instance_new.GetBufferPoolManager(), samehada_instance_new.GetLogManager(), samehada_instance_new.GetLockManager(), txn_new)

	columnToCheck := catalog_recov.GetTableByOID(1).Schema().GetColumn(1)

	testingpkg.Assert(t, columnToCheck.GetColumnName() == "b", "")
	testingpkg.Assert(t, columnToCheck.GetType() == 4, "")
	testingpkg.Assert(t, columnToCheck.HasIndex() == true, "")

	samehada_instance.Finalize(true)
}
