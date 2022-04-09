// this code is from https://github.com/brunocalza/go-bustub
// there is license and copyright notice in licenses/go-bustub dir

package executors

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/ryogrid/SamehadaDB/catalog"
	"github.com/ryogrid/SamehadaDB/common"
	"github.com/ryogrid/SamehadaDB/execution/expression"
	"github.com/ryogrid/SamehadaDB/execution/plans"
	"github.com/ryogrid/SamehadaDB/recovery"
	"github.com/ryogrid/SamehadaDB/storage/access"
	"github.com/ryogrid/SamehadaDB/storage/buffer"
	"github.com/ryogrid/SamehadaDB/storage/disk"
	"github.com/ryogrid/SamehadaDB/storage/table/column"
	"github.com/ryogrid/SamehadaDB/storage/table/schema"
	"github.com/ryogrid/SamehadaDB/storage/tuple"
	testingpkg "github.com/ryogrid/SamehadaDB/testing"
	"github.com/ryogrid/SamehadaDB/types"
)

func TestSimpleInsertAndSeqScan(t *testing.T) {
	diskManager := disk.NewDiskManagerTest()
	defer diskManager.ShutDown()
	log_mgr := recovery.NewLogManager(&diskManager)
	bpm := buffer.NewBufferPoolManager(uint32(32), diskManager, log_mgr)
	txn_mgr := access.NewTransactionManager(log_mgr)
	txn := txn_mgr.Begin(nil)

	c := catalog.BootstrapCatalog(bpm, log_mgr, access.NewLockManager(access.REGULAR, access.PREVENTION), txn)

	columnA := column.NewColumn("a", types.Integer, false)
	columnB := column.NewColumn("b", types.Integer, false)
	schema_ := schema.NewSchema([]*column.Column{columnA, columnB})

	tableMetadata := c.CreateTable("test_1", schema_, txn)

	row1 := make([]types.Value, 0)
	row1 = append(row1, types.NewInteger(20))
	row1 = append(row1, types.NewInteger(22))

	row2 := make([]types.Value, 0)
	row2 = append(row2, types.NewInteger(99))
	row2 = append(row2, types.NewInteger(55))

	rows := make([][]types.Value, 0)
	rows = append(rows, row1)
	rows = append(rows, row2)

	insertPlanNode := plans.NewInsertPlanNode(rows, tableMetadata.OID())

	executionEngine := &ExecutionEngine{}
	executorContext := NewExecutorContext(c, bpm, txn)
	executionEngine.Execute(insertPlanNode, executorContext)

	bpm.FlushAllPages()

	outColumnA := column.NewColumn("a", types.Integer, false)
	outSchema := schema.NewSchema([]*column.Column{outColumnA})

	seqPlan := plans.NewSeqScanPlanNode(outSchema, nil, tableMetadata.OID())

	results := executionEngine.Execute(seqPlan, executorContext)

	txn_mgr.Commit(txn)

	testingpkg.Assert(t, types.NewInteger(20).CompareEquals(results[0].GetValue(outSchema, 0)), "value should be 20")
	testingpkg.Assert(t, types.NewInteger(99).CompareEquals(results[1].GetValue(outSchema, 0)), "value should be 99")
}

func TestSimpleInsertAndSeqScanWithPredicateComparison(t *testing.T) {
	diskManager := disk.NewDiskManagerTest()
	defer diskManager.ShutDown()
	log_mgr := recovery.NewLogManager(&diskManager)
	bpm := buffer.NewBufferPoolManager(uint32(32), diskManager, log_mgr) //, recovery.NewLogManager(diskManager), access.NewLockManager(access.REGULAR, access.PREVENTION))
	txn_mgr := access.NewTransactionManager(log_mgr)
	txn := txn_mgr.Begin(nil)

	c := catalog.BootstrapCatalog(bpm, log_mgr, access.NewLockManager(access.REGULAR, access.PREVENTION), txn)

	columnA := column.NewColumn("a", types.Integer, false)
	columnB := column.NewColumn("b", types.Integer, false)
	columnC := column.NewColumn("c", types.Varchar, false)
	schema_ := schema.NewSchema([]*column.Column{columnA, columnB, columnC})

	tableMetadata := c.CreateTable("test_1", schema_, txn)

	row1 := make([]types.Value, 0)
	row1 = append(row1, types.NewInteger(20))
	row1 = append(row1, types.NewInteger(22))
	row1 = append(row1, types.NewVarchar("foo"))

	row2 := make([]types.Value, 0)
	row2 = append(row2, types.NewInteger(99))
	row2 = append(row2, types.NewInteger(55))
	row2 = append(row2, types.NewVarchar("bar"))

	rows := make([][]types.Value, 0)
	rows = append(rows, row1)
	rows = append(rows, row2)

	insertPlanNode := plans.NewInsertPlanNode(rows, tableMetadata.OID())

	executionEngine := &ExecutionEngine{}
	executorContext := NewExecutorContext(c, bpm, txn)
	executionEngine.Execute(insertPlanNode, executorContext)

	bpm.FlushAllPages()

	txn_mgr.Commit(txn)

	cases := []SeqScanTestCase{{
		"select a ... WHERE b = 55",
		executionEngine,
		executorContext,
		tableMetadata,
		[]Column{{"a", types.Integer}},
		Predicate{"b", expression.Equal, 55},
		[]Assertion{{"a", 99}},
		1,
	}, {
		"select b ... WHERE b = 55",
		executionEngine,
		executorContext,
		tableMetadata,
		[]Column{{"b", types.Integer}},
		Predicate{"b", expression.Equal, 55},
		[]Assertion{{"b", 55}},
		1,
	}, {
		"select a, b ... WHERE a = 20",
		executionEngine,
		executorContext,
		tableMetadata,
		[]Column{{"a", types.Integer}, {"b", types.Integer}},
		Predicate{"a", expression.Equal, 20},
		[]Assertion{{"a", 20}, {"b", 22}},
		1,
	}, {
		"select a, b ... WHERE a = 99",
		executionEngine,
		executorContext,
		tableMetadata,
		[]Column{{"a", types.Integer}, {"b", types.Integer}},
		Predicate{"a", expression.Equal, 99},
		[]Assertion{{"a", 99}, {"b", 55}},
		1,
	}, {
		"select a, b ... WHERE a = 100",
		executionEngine,
		executorContext,
		tableMetadata,
		[]Column{{"a", types.Integer}, {"b", types.Integer}},
		Predicate{"a", expression.Equal, 100},
		[]Assertion{},
		0,
	}, {
		"select a, b ... WHERE b != 55",
		executionEngine,
		executorContext,
		tableMetadata,
		[]Column{{"a", types.Integer}, {"b", types.Integer}},
		Predicate{"b", expression.NotEqual, 55},
		[]Assertion{{"a", 20}, {"b", 22}},
		1,
	}, {
		"select a, b, c ... WHERE c = 'foo'",
		executionEngine,
		executorContext,
		tableMetadata,
		[]Column{{"a", types.Integer}, {"b", types.Integer}, {"c", types.Varchar}},
		Predicate{"c", expression.Equal, "foo"},
		[]Assertion{{"a", 20}, {"b", 22}, {"c", "foo"}},
		1,
	}, {
		"select a, b, c ... WHERE c != 'foo'",
		executionEngine,
		executorContext,
		tableMetadata,
		[]Column{{"a", types.Integer}, {"b", types.Integer}, {"c", types.Varchar}},
		Predicate{"c", expression.NotEqual, "foo"},
		[]Assertion{{"a", 99}, {"b", 55}, {"c", "bar"}},
		1,
	}}

	for _, test := range cases {
		t.Run(test.Description, func(t *testing.T) {
			ExecuteSeqScanTestCase(t, test)
		})
	}
}

func TestSimpleInsertAndLimitExecution(t *testing.T) {
	diskManager := disk.NewDiskManagerTest()
	defer diskManager.ShutDown()
	log_mgr := recovery.NewLogManager(&diskManager)
	bpm := buffer.NewBufferPoolManager(uint32(32), diskManager, log_mgr)
	txn_mgr := access.NewTransactionManager(log_mgr)
	txn := txn_mgr.Begin(nil)

	c := catalog.BootstrapCatalog(bpm, log_mgr, access.NewLockManager(access.REGULAR, access.PREVENTION), txn)

	columnA := column.NewColumn("a", types.Integer, false)
	columnB := column.NewColumn("b", types.Integer, false)
	schema_ := schema.NewSchema([]*column.Column{columnA, columnB})

	tableMetadata := c.CreateTable("test_1", schema_, txn)

	row1 := make([]types.Value, 0)
	row1 = append(row1, types.NewInteger(20))
	row1 = append(row1, types.NewInteger(22))

	row2 := make([]types.Value, 0)
	row2 = append(row2, types.NewInteger(99))
	row2 = append(row2, types.NewInteger(55))

	row3 := make([]types.Value, 0)
	row3 = append(row3, types.NewInteger(11))
	row3 = append(row3, types.NewInteger(44))

	row4 := make([]types.Value, 0)
	row4 = append(row4, types.NewInteger(76))
	row4 = append(row4, types.NewInteger(90))

	rows := make([][]types.Value, 0)
	rows = append(rows, row1)
	rows = append(rows, row2)
	rows = append(rows, row3)
	rows = append(rows, row4)

	insertPlanNode := plans.NewInsertPlanNode(rows, tableMetadata.OID())

	executionEngine := &ExecutionEngine{}
	executorContext := NewExecutorContext(c, bpm, txn)
	executionEngine.Execute(insertPlanNode, executorContext)

	bpm.FlushAllPages()

	txn_mgr.Commit(txn)

	// TEST 1: select a, b ... LIMIT 1
	func() {
		a := column.NewColumn("a", types.Integer, false)
		b := column.NewColumn("b", types.Integer, false)
		outSchema := schema.NewSchema([]*column.Column{a, b})
		seqPlan := plans.NewSeqScanPlanNode(outSchema, nil, tableMetadata.OID())
		limitPlan := plans.NewLimitPlanNode(seqPlan, 1, 1)

		results := executionEngine.Execute(limitPlan, executorContext)

		testingpkg.Equals(t, 1, len(results))
		testingpkg.Assert(t, types.NewInteger(99).CompareEquals(results[0].GetValue(outSchema, 0)), "value should be 99 but was %d", results[0].GetValue(outSchema, 0).ToInteger())
		testingpkg.Assert(t, types.NewInteger(55).CompareEquals(results[0].GetValue(outSchema, 1)), "value should be 55 but was %d", results[0].GetValue(outSchema, 1).ToInteger())
	}()

	// TEST 1: select a, b ... LIMIT 2
	func() {
		a := column.NewColumn("a", types.Integer, false)
		b := column.NewColumn("b", types.Integer, false)
		outSchema := schema.NewSchema([]*column.Column{a, b})
		seqPlan := plans.NewSeqScanPlanNode(outSchema, nil, tableMetadata.OID())
		limitPlan := plans.NewLimitPlanNode(seqPlan, 2, 0)

		results := executionEngine.Execute(limitPlan, executorContext)

		testingpkg.Equals(t, 2, len(results))
	}()

	// TEST 1: select a, b ... LIMIT 3
	func() {
		a := column.NewColumn("a", types.Integer, false)
		b := column.NewColumn("b", types.Integer, false)
		outSchema := schema.NewSchema([]*column.Column{a, b})
		seqPlan := plans.NewSeqScanPlanNode(outSchema, nil, tableMetadata.OID())
		limitPlan := plans.NewLimitPlanNode(seqPlan, 3, 0)

		results := executionEngine.Execute(limitPlan, executorContext)

		testingpkg.Equals(t, 3, len(results))
	}()
}

func TestSimpleInsertAndLimitExecutionMultiTable(t *testing.T) {
	diskManager := disk.NewDiskManagerTest()
	defer diskManager.ShutDown()
	log_mgr := recovery.NewLogManager(&diskManager)
	bpm := buffer.NewBufferPoolManager(uint32(32), diskManager, log_mgr)
	txn_mgr := access.NewTransactionManager(log_mgr)
	txn := txn_mgr.Begin(nil)

	c := catalog.BootstrapCatalog(bpm, log_mgr, access.NewLockManager(access.REGULAR, access.PREVENTION), txn)

	columnA := column.NewColumn("a", types.Integer, false)
	columnB := column.NewColumn("b", types.Integer, false)
	schema_ := schema.NewSchema([]*column.Column{columnA, columnB})

	tableMetadata := c.CreateTable("test_1", schema_, txn)

	row1 := make([]types.Value, 0)
	row1 = append(row1, types.NewInteger(20))
	row1 = append(row1, types.NewInteger(22))

	row2 := make([]types.Value, 0)
	row2 = append(row2, types.NewInteger(99))
	row2 = append(row2, types.NewInteger(55))

	row3 := make([]types.Value, 0)
	row3 = append(row3, types.NewInteger(11))
	row3 = append(row3, types.NewInteger(44))

	row4 := make([]types.Value, 0)
	row4 = append(row4, types.NewInteger(76))
	row4 = append(row4, types.NewInteger(90))

	rows := make([][]types.Value, 0)
	rows = append(rows, row1)
	rows = append(rows, row2)
	rows = append(rows, row3)
	rows = append(rows, row4)

	insertPlanNode := plans.NewInsertPlanNode(rows, tableMetadata.OID())

	executionEngine := &ExecutionEngine{}
	executorContext := NewExecutorContext(c, bpm, txn)
	executionEngine.Execute(insertPlanNode, executorContext)

	// construct second table

	columnA = column.NewColumn("a", types.Integer, false)
	columnB = column.NewColumn("b", types.Integer, false)
	schema_ = schema.NewSchema([]*column.Column{columnA, columnB})

	tableMetadata2 := c.CreateTable("test_2", schema_, txn)

	row1 = make([]types.Value, 0)
	row1 = append(row1, types.NewInteger(20))
	row1 = append(row1, types.NewInteger(22))

	row2 = make([]types.Value, 0)
	row2 = append(row2, types.NewInteger(99))
	row2 = append(row2, types.NewInteger(55))

	row3 = make([]types.Value, 0)
	row3 = append(row3, types.NewInteger(11))
	row3 = append(row3, types.NewInteger(44))

	row4 = make([]types.Value, 0)
	row4 = append(row4, types.NewInteger(76))
	row4 = append(row4, types.NewInteger(90))

	rows = make([][]types.Value, 0)
	rows = append(rows, row1)
	rows = append(rows, row2)
	rows = append(rows, row3)
	rows = append(rows, row4)

	insertPlanNode = plans.NewInsertPlanNode(rows, tableMetadata2.OID())

	//executionEngine := &ExecutionEngine{}
	//executorContext := NewExecutorContext(c, bpm, txn)
	executionEngine.Execute(insertPlanNode, executorContext)

	bpm.FlushAllPages()

	txn_mgr.Commit(txn)

	// TEST 1: select a, b ... LIMIT 1
	func() {
		a := column.NewColumn("a", types.Integer, false)
		b := column.NewColumn("b", types.Integer, false)
		outSchema := schema.NewSchema([]*column.Column{a, b})
		seqPlan := plans.NewSeqScanPlanNode(outSchema, nil, tableMetadata.OID())
		limitPlan := plans.NewLimitPlanNode(seqPlan, 1, 1)

		results := executionEngine.Execute(limitPlan, executorContext)

		testingpkg.Equals(t, 1, len(results))
		testingpkg.Assert(t, types.NewInteger(99).CompareEquals(results[0].GetValue(outSchema, 0)), "value should be 99 but was %d", results[0].GetValue(outSchema, 0).ToInteger())
		testingpkg.Assert(t, types.NewInteger(55).CompareEquals(results[0].GetValue(outSchema, 1)), "value should be 55 but was %d", results[0].GetValue(outSchema, 1).ToInteger())
	}()

	// TEST 1: select a, b ... LIMIT 2
	func() {
		a := column.NewColumn("a", types.Integer, false)
		b := column.NewColumn("b", types.Integer, false)
		outSchema := schema.NewSchema([]*column.Column{a, b})
		seqPlan := plans.NewSeqScanPlanNode(outSchema, nil, tableMetadata.OID())
		limitPlan := plans.NewLimitPlanNode(seqPlan, 2, 0)

		results := executionEngine.Execute(limitPlan, executorContext)

		testingpkg.Equals(t, 2, len(results))
	}()

	// TEST 1: select a, b ... LIMIT 1
	func() {
		a := column.NewColumn("a", types.Integer, false)
		b := column.NewColumn("b", types.Integer, false)
		outSchema := schema.NewSchema([]*column.Column{a, b})
		seqPlan := plans.NewSeqScanPlanNode(outSchema, nil, tableMetadata2.OID())
		limitPlan := plans.NewLimitPlanNode(seqPlan, 1, 1)

		results := executionEngine.Execute(limitPlan, executorContext)

		testingpkg.Equals(t, 1, len(results))
		testingpkg.Assert(t, types.NewInteger(99).CompareEquals(results[0].GetValue(outSchema, 0)), "value should be 99 but was %d", results[0].GetValue(outSchema, 0).ToInteger())
		testingpkg.Assert(t, types.NewInteger(55).CompareEquals(results[0].GetValue(outSchema, 1)), "value should be 55 but was %d", results[0].GetValue(outSchema, 1).ToInteger())
	}()

	// TEST 1: select a, b ... LIMIT 3
	func() {
		a := column.NewColumn("a", types.Integer, false)
		b := column.NewColumn("b", types.Integer, false)
		outSchema := schema.NewSchema([]*column.Column{a, b})
		seqPlan := plans.NewSeqScanPlanNode(outSchema, nil, tableMetadata2.OID())
		limitPlan := plans.NewLimitPlanNode(seqPlan, 3, 0)

		results := executionEngine.Execute(limitPlan, executorContext)

		testingpkg.Equals(t, 3, len(results))
	}()
}

func TestHashTableIndex(t *testing.T) {
	diskManager := disk.NewDiskManagerTest()
	defer diskManager.ShutDown()
	log_mgr := recovery.NewLogManager(&diskManager)
	bpm := buffer.NewBufferPoolManager(uint32(32), diskManager, log_mgr) //, recovery.NewLogManager(diskManager), access.NewLockManager(access.REGULAR, access.PREVENTION))
	// TODO: (SDB) [logging/recovery] need increment of transaction ID

	txn_mgr := access.NewTransactionManager(log_mgr)
	txn := txn_mgr.Begin(nil)

	c := catalog.BootstrapCatalog(bpm, log_mgr, access.NewLockManager(access.REGULAR, access.PREVENTION), txn)

	columnA := column.NewColumn("a", types.Integer, true)
	columnB := column.NewColumn("b", types.Integer, true)
	columnC := column.NewColumn("c", types.Varchar, true)
	schema_ := schema.NewSchema([]*column.Column{columnA, columnB, columnC})

	tableMetadata := c.CreateTable("test_1", schema_, txn)

	row1 := make([]types.Value, 0)
	row1 = append(row1, types.NewInteger(20))
	row1 = append(row1, types.NewInteger(22))
	row1 = append(row1, types.NewVarchar("foo"))

	row2 := make([]types.Value, 0)
	row2 = append(row2, types.NewInteger(99))
	row2 = append(row2, types.NewInteger(55))
	row2 = append(row2, types.NewVarchar("bar"))

	row3 := make([]types.Value, 0)
	row3 = append(row3, types.NewInteger(1225))
	row3 = append(row3, types.NewInteger(712))
	row3 = append(row3, types.NewVarchar("baz"))

	row4 := make([]types.Value, 0)
	row4 = append(row4, types.NewInteger(1225))
	row4 = append(row4, types.NewInteger(712))
	row4 = append(row4, types.NewVarchar("baz"))

	rows := make([][]types.Value, 0)
	rows = append(rows, row1)
	rows = append(rows, row2)
	rows = append(rows, row3)
	rows = append(rows, row4)

	insertPlanNode := plans.NewInsertPlanNode(rows, tableMetadata.OID())

	executionEngine := &ExecutionEngine{}
	executorContext := NewExecutorContext(c, bpm, txn)
	executionEngine.Execute(insertPlanNode, executorContext)

	bpm.FlushAllPages()

	txn_mgr.Commit(txn)

	cases := []HashIndexScanTestCase{{
		"select a ... WHERE b = 55",
		executionEngine,
		executorContext,
		tableMetadata,
		[]Column{{"a", types.Integer}},
		Predicate{"b", expression.Equal, 55},
		[]Assertion{{"a", 99}},
		1,
	}, {
		"select b ... WHERE b = 55",
		executionEngine,
		executorContext,
		tableMetadata,
		[]Column{{"b", types.Integer}},
		Predicate{"b", expression.Equal, 55},
		[]Assertion{{"b", 55}},
		1,
	}, {
		"select a, b ... WHERE a = 20",
		executionEngine,
		executorContext,
		tableMetadata,
		[]Column{{"a", types.Integer}, {"b", types.Integer}},
		Predicate{"a", expression.Equal, 20},
		[]Assertion{{"a", 20}, {"b", 22}},
		1,
	}, {
		"select a, b ... WHERE a = 99",
		executionEngine,
		executorContext,
		tableMetadata,
		[]Column{{"a", types.Integer}, {"b", types.Integer}},
		Predicate{"a", expression.Equal, 99},
		[]Assertion{{"a", 99}, {"b", 55}},
		1,
	}, {
		"select a, b ... WHERE a = 100",
		executionEngine,
		executorContext,
		tableMetadata,
		[]Column{{"a", types.Integer}, {"b", types.Integer}},
		Predicate{"a", expression.Equal, 100},
		[]Assertion{},
		0,
	}, {
		"select a, b ... WHERE b = 55",
		executionEngine,
		executorContext,
		tableMetadata,
		[]Column{{"a", types.Integer}, {"b", types.Integer}},
		Predicate{"b", expression.Equal, 55},
		[]Assertion{{"a", 99}, {"b", 55}},
		1,
	}, {
		"select a, b, c ... WHERE c = 'foo'",
		executionEngine,
		executorContext,
		tableMetadata,
		[]Column{{"a", types.Integer}, {"b", types.Integer}, {"c", types.Varchar}},
		Predicate{"c", expression.Equal, "foo"},
		[]Assertion{{"a", 20}, {"b", 22}, {"c", "foo"}},
		1,
	}, {
		"select a, b ... WHERE c = 'baz'",
		executionEngine,
		executorContext,
		tableMetadata,
		[]Column{{"a", types.Integer}, {"b", types.Integer}},
		Predicate{"c", expression.Equal, "baz"},
		[]Assertion{{"a", 1225}, {"b", 712}},
		2,
	}}

	for _, test := range cases {
		t.Run(test.Description, func(t *testing.T) {
			ExecuteHashIndexScanTestCase(t, test)
		})
	}

}

func TestSimpleDelete(t *testing.T) {
	diskManager := disk.NewDiskManagerTest()
	defer diskManager.ShutDown()
	log_mgr := recovery.NewLogManager(&diskManager)
	bpm := buffer.NewBufferPoolManager(uint32(32), diskManager, log_mgr)
	txn_mgr := access.NewTransactionManager(log_mgr)
	txn := txn_mgr.Begin(nil)

	c := catalog.BootstrapCatalog(bpm, log_mgr, access.NewLockManager(access.REGULAR, access.PREVENTION), txn)

	columnA := column.NewColumn("a", types.Integer, false)
	columnB := column.NewColumn("b", types.Integer, false)
	columnC := column.NewColumn("c", types.Varchar, false)
	schema_ := schema.NewSchema([]*column.Column{columnA, columnB, columnC})

	tableMetadata := c.CreateTable("test_1", schema_, txn)

	row1 := make([]types.Value, 0)
	row1 = append(row1, types.NewInteger(20))
	row1 = append(row1, types.NewInteger(22))
	row1 = append(row1, types.NewVarchar("foo"))

	row2 := make([]types.Value, 0)
	row2 = append(row2, types.NewInteger(99))
	row2 = append(row2, types.NewInteger(55))
	row2 = append(row2, types.NewVarchar("bar"))

	row3 := make([]types.Value, 0)
	row3 = append(row3, types.NewInteger(1225))
	row3 = append(row3, types.NewInteger(712))
	row3 = append(row3, types.NewVarchar("baz"))

	row4 := make([]types.Value, 0)
	row4 = append(row4, types.NewInteger(1225))
	row4 = append(row4, types.NewInteger(712))
	row4 = append(row4, types.NewVarchar("baz"))

	rows := make([][]types.Value, 0)
	rows = append(rows, row1)
	rows = append(rows, row2)
	rows = append(rows, row3)
	rows = append(rows, row4)

	insertPlanNode := plans.NewInsertPlanNode(rows, tableMetadata.OID())

	executionEngine := &ExecutionEngine{}
	executorContext := NewExecutorContext(c, bpm, txn)
	executionEngine.Execute(insertPlanNode, executorContext)

	bpm.FlushAllPages()

	txn_mgr.Commit(txn)

	cases := []DeleteTestCase{{
		"delete ... WHERE c = 'baz'",
		txn_mgr,
		executionEngine,
		executorContext,
		tableMetadata,
		[]Column{{"a", types.Integer}, {"b", types.Integer}, {"c", types.Varchar}},
		Predicate{"c", expression.Equal, "baz"},
		[]Assertion{{"a", 1225}, {"b", 712}, {"c", "baz"}},
		2,
	}, {
		"delete ... WHERE b = 55",
		txn_mgr,
		executionEngine,
		executorContext,
		tableMetadata,
		[]Column{{"a", types.Integer}, {"b", types.Integer}, {"c", types.Varchar}},
		Predicate{"b", expression.Equal, 55},
		[]Assertion{{"a", 99}, {"b", 55}, {"c", "bar"}},
		1,
	}, {
		"delete ... WHERE a = 20",
		txn_mgr,
		executionEngine,
		executorContext,
		tableMetadata,
		[]Column{{"a", types.Integer}, {"b", types.Integer}, {"c", types.Varchar}},
		Predicate{"a", expression.Equal, 20},
		[]Assertion{{"a", 20}, {"b", 22}, {"c", "foo"}},
		1,
	}}

	for _, test := range cases {
		t.Run(test.Description, func(t *testing.T) {
			ExecuteDeleteTestCase(t, test)
		})
	}
}

func TestDeleteWithSelctInsert(t *testing.T) {
	diskManager := disk.NewDiskManagerTest()
	defer diskManager.ShutDown()
	log_mgr := recovery.NewLogManager(&diskManager)
	bpm := buffer.NewBufferPoolManager(uint32(32), diskManager, log_mgr)
	txn_mgr := access.NewTransactionManager(log_mgr)
	txn := txn_mgr.Begin(nil)

	c := catalog.BootstrapCatalog(bpm, log_mgr, access.NewLockManager(access.REGULAR, access.PREVENTION), txn)

	columnA := column.NewColumn("a", types.Integer, false)
	columnB := column.NewColumn("b", types.Integer, false)
	columnC := column.NewColumn("c", types.Varchar, false)
	schema_ := schema.NewSchema([]*column.Column{columnA, columnB, columnC})

	tableMetadata := c.CreateTable("test_1", schema_, txn)

	row1 := make([]types.Value, 0)
	row1 = append(row1, types.NewInteger(20))
	row1 = append(row1, types.NewInteger(22))
	row1 = append(row1, types.NewVarchar("foo"))

	row2 := make([]types.Value, 0)
	row2 = append(row2, types.NewInteger(99))
	row2 = append(row2, types.NewInteger(55))
	row2 = append(row2, types.NewVarchar("bar"))

	row3 := make([]types.Value, 0)
	row3 = append(row3, types.NewInteger(1225))
	row3 = append(row3, types.NewInteger(712))
	row3 = append(row3, types.NewVarchar("baz"))

	row4 := make([]types.Value, 0)
	row4 = append(row4, types.NewInteger(1225))
	row4 = append(row4, types.NewInteger(712))
	row4 = append(row4, types.NewVarchar("baz"))

	rows := make([][]types.Value, 0)
	rows = append(rows, row1)
	rows = append(rows, row2)
	rows = append(rows, row3)
	rows = append(rows, row4)

	insertPlanNode := plans.NewInsertPlanNode(rows, tableMetadata.OID())

	executionEngine := &ExecutionEngine{}
	executorContext := NewExecutorContext(c, bpm, txn)
	executionEngine.Execute(insertPlanNode, executorContext)

	bpm.FlushAllPages()

	txn_mgr.Commit(txn)

	cases := []DeleteTestCase{{
		"delete ... WHERE c = 'baz'",
		txn_mgr,
		executionEngine,
		executorContext,
		tableMetadata,
		[]Column{{"a", types.Integer}, {"b", types.Integer}, {"c", types.Varchar}},
		Predicate{"c", expression.Equal, "baz"},
		[]Assertion{{"a", 1225}, {"b", 712}, {"c", "baz"}},
		2,
	}}

	for _, test := range cases {
		t.Run(test.Description, func(t *testing.T) {
			ExecuteDeleteTestCase(t, test)
		})
	}

	cases2 := []SeqScanTestCase{{
		"select a ... WHERE c = baz",
		executionEngine,
		executorContext,
		tableMetadata,
		[]Column{{"a", types.Integer}},
		Predicate{"c", expression.Equal, "baz"},
		[]Assertion{{"a", 99}},
		0,
	}, {
		"select b ... WHERE b = 55",
		executionEngine,
		executorContext,
		tableMetadata,
		[]Column{{"b", types.Integer}},
		Predicate{"b", expression.Equal, 55},
		[]Assertion{{"b", 55}},
		1,
	}}

	for _, test := range cases2 {
		t.Run(test.Description, func(t *testing.T) {
			ExecuteSeqScanTestCase(t, test)
		})
	}

	// insert new records
	txn = txn_mgr.Begin(nil)
	row1 = make([]types.Value, 0)
	row1 = append(row1, types.NewInteger(666))
	row1 = append(row1, types.NewInteger(777))
	row1 = append(row1, types.NewVarchar("fin"))
	rows = make([][]types.Value, 0)
	rows = append(rows, row1)

	insertPlanNode = plans.NewInsertPlanNode(rows, tableMetadata.OID())

	executionEngine = &ExecutionEngine{}
	executorContext = NewExecutorContext(c, bpm, txn)
	executionEngine.Execute(insertPlanNode, executorContext)
	bpm.FlushAllPages()
	txn_mgr.Commit(txn)

	cases3 := []SeqScanTestCase{{
		"select a,c ... WHERE b = 777",
		executionEngine,
		executorContext,
		tableMetadata,
		[]Column{{"a", types.Integer}, {"c", types.Varchar}},
		Predicate{"b", expression.Equal, 777},
		[]Assertion{{"a", 666}, {"c", "fin"}},
		1,
	}}

	for _, test := range cases3 {
		t.Run(test.Description, func(t *testing.T) {
			ExecuteSeqScanTestCase(t, test)
		})
	}
}

func TestSimpleInsertAndUpdate(t *testing.T) {
	diskManager := disk.NewDiskManagerTest()
	defer diskManager.ShutDown()
	log_mgr := recovery.NewLogManager(&diskManager)
	bpm := buffer.NewBufferPoolManager(uint32(32), diskManager, log_mgr)
	txn_mgr := access.NewTransactionManager(log_mgr)
	txn := txn_mgr.Begin(nil)

	c := catalog.BootstrapCatalog(bpm, log_mgr, access.NewLockManager(access.REGULAR, access.PREVENTION), txn)

	columnA := column.NewColumn("a", types.Integer, false)
	columnB := column.NewColumn("b", types.Varchar, false)
	schema_ := schema.NewSchema([]*column.Column{columnA, columnB})

	tableMetadata := c.CreateTable("test_1", schema_, txn)

	row1 := make([]types.Value, 0)
	row1 = append(row1, types.NewInteger(20))
	row1 = append(row1, types.NewVarchar("hoge"))

	row2 := make([]types.Value, 0)
	row2 = append(row2, types.NewInteger(99))
	row2 = append(row2, types.NewVarchar("foo"))

	rows := make([][]types.Value, 0)
	rows = append(rows, row1)
	rows = append(rows, row2)

	insertPlanNode := plans.NewInsertPlanNode(rows, tableMetadata.OID())

	executionEngine := &ExecutionEngine{}
	executorContext := NewExecutorContext(c, bpm, txn)
	executionEngine.Execute(insertPlanNode, executorContext)

	txn_mgr.Commit(txn)

	fmt.Println("update a row...")
	txn = txn_mgr.Begin(nil)

	row1 = make([]types.Value, 0)
	row1 = append(row1, types.NewInteger(99))
	row1 = append(row1, types.NewVarchar("updated"))

	pred := Predicate{"b", expression.Equal, "foo"}
	tmpColVal := new(expression.ColumnValue)
	tmpColVal.SetTupleIndex(0)
	tmpColVal.SetColIndex(tableMetadata.Schema().GetColIndex(pred.LeftColumn))
	expression_ := expression.NewComparison(*tmpColVal, expression.NewConstantValue(GetValue(pred.RightColumn), GetValueType(pred.LeftColumn)), pred.Operator, types.Boolean)

	updatePlanNode := plans.NewUpdatePlanNode(row1, &expression_, tableMetadata.OID())
	executionEngine.Execute(updatePlanNode, executorContext)

	txn_mgr.Commit(txn)

	fmt.Println("select and check value...")
	txn = txn_mgr.Begin(nil)

	outColumnB := column.NewColumn("b", types.Varchar, false)
	outSchema := schema.NewSchema([]*column.Column{outColumnB})

	pred = Predicate{"a", expression.Equal, 99}
	tmpColVal = new(expression.ColumnValue)
	tmpColVal.SetTupleIndex(0)
	tmpColVal.SetColIndex(tableMetadata.Schema().GetColIndex(pred.LeftColumn))
	expression_ = expression.NewComparison(*tmpColVal, expression.NewConstantValue(GetValue(pred.RightColumn), GetValueType(pred.RightColumn)), pred.Operator, types.Boolean)

	seqPlan := plans.NewSeqScanPlanNode(outSchema, &expression_, tableMetadata.OID())
	results := executionEngine.Execute(seqPlan, executorContext)

	txn_mgr.Commit(txn)

	testingpkg.Assert(t, types.NewVarchar("updated").CompareEquals(results[0].GetValue(outSchema, 0)), "value should be 'updated'")
	//testingpkg.Assert(t, types.NewInteger(99).CompareEquals(results[1].GetValue(outSchema, 0)), "value should be 99")
}

func TestInsertUpdateMix(t *testing.T) {
	diskManager := disk.NewDiskManagerTest()
	defer diskManager.ShutDown()
	log_mgr := recovery.NewLogManager(&diskManager)
	bpm := buffer.NewBufferPoolManager(uint32(32), diskManager, log_mgr)
	txn_mgr := access.NewTransactionManager(log_mgr)
	txn := txn_mgr.Begin(nil)

	c := catalog.BootstrapCatalog(bpm, log_mgr, access.NewLockManager(access.REGULAR, access.PREVENTION), txn)

	columnA := column.NewColumn("a", types.Integer, false)
	columnB := column.NewColumn("b", types.Varchar, false)
	schema_ := schema.NewSchema([]*column.Column{columnA, columnB})

	tableMetadata := c.CreateTable("test_1", schema_, txn)

	row1 := make([]types.Value, 0)
	row1 = append(row1, types.NewInteger(20))
	row1 = append(row1, types.NewVarchar("hoge"))

	row2 := make([]types.Value, 0)
	row2 = append(row2, types.NewInteger(99))
	row2 = append(row2, types.NewVarchar("foo"))

	rows := make([][]types.Value, 0)
	rows = append(rows, row1)
	rows = append(rows, row2)

	insertPlanNode := plans.NewInsertPlanNode(rows, tableMetadata.OID())

	executionEngine := &ExecutionEngine{}
	executorContext := NewExecutorContext(c, bpm, txn)
	executionEngine.Execute(insertPlanNode, executorContext)

	txn_mgr.Commit(txn)

	fmt.Println("update a row...")
	txn = txn_mgr.Begin(nil)

	row1 = make([]types.Value, 0)
	row1 = append(row1, types.NewInteger(99))
	row1 = append(row1, types.NewVarchar("updated"))

	pred := Predicate{"b", expression.Equal, "foo"}
	tmpColVal := new(expression.ColumnValue)
	tmpColVal.SetTupleIndex(0)
	tmpColVal.SetColIndex(tableMetadata.Schema().GetColIndex(pred.LeftColumn))
	expression_ := expression.NewComparison(*tmpColVal, expression.NewConstantValue(GetValue(pred.RightColumn), GetValueType(pred.RightColumn)), pred.Operator, types.Boolean)

	updatePlanNode := plans.NewUpdatePlanNode(row1, &expression_, tableMetadata.OID())
	executionEngine.Execute(updatePlanNode, executorContext)

	txn_mgr.Commit(txn)

	fmt.Println("select and check value...")
	txn = txn_mgr.Begin(nil)

	outColumnB := column.NewColumn("b", types.Varchar, false)
	outSchema := schema.NewSchema([]*column.Column{outColumnB})

	pred = Predicate{"a", expression.Equal, 99}
	tmpColVal = new(expression.ColumnValue)
	tmpColVal.SetTupleIndex(0)
	tmpColVal.SetColIndex(tableMetadata.Schema().GetColIndex(pred.LeftColumn))
	expression_ = expression.NewComparison(*tmpColVal, expression.NewConstantValue(GetValue(pred.RightColumn), GetValueType(pred.RightColumn)), pred.Operator, types.Boolean)

	seqPlan := plans.NewSeqScanPlanNode(outSchema, &expression_, tableMetadata.OID())
	results := executionEngine.Execute(seqPlan, executorContext)

	txn_mgr.Commit(txn)

	testingpkg.Assert(t, types.NewVarchar("updated").CompareEquals(results[0].GetValue(outSchema, 0)), "value should be 'updated'")

	fmt.Println("insert after update...")

	txn = txn_mgr.Begin(nil)

	row1 = make([]types.Value, 0)
	row1 = append(row1, types.NewInteger(77))
	row1 = append(row1, types.NewVarchar("hage"))

	row2 = make([]types.Value, 0)
	row2 = append(row2, types.NewInteger(666))
	row2 = append(row2, types.NewVarchar("fuba"))

	rows = make([][]types.Value, 0)
	rows = append(rows, row1)
	rows = append(rows, row2)

	insertPlanNode = plans.NewInsertPlanNode(rows, tableMetadata.OID())

	executionEngine.Execute(insertPlanNode, executorContext)

	txn_mgr.Commit(txn)

	fmt.Println("select inserted row and check value...")
	txn = txn_mgr.Begin(nil)

	outColumnA := column.NewColumn("a", types.Integer, false)
	outSchema = schema.NewSchema([]*column.Column{outColumnA})

	pred = Predicate{"b", expression.Equal, "hage"}
	tmpColVal = new(expression.ColumnValue)
	tmpColVal.SetTupleIndex(0)
	tmpColVal.SetColIndex(tableMetadata.Schema().GetColIndex(pred.LeftColumn))
	expression_ = expression.NewComparison(*tmpColVal, expression.NewConstantValue(GetValue(pred.RightColumn), GetValueType(pred.RightColumn)), pred.Operator, types.Boolean)

	seqPlan = plans.NewSeqScanPlanNode(outSchema, &expression_, tableMetadata.OID())
	results = executionEngine.Execute(seqPlan, executorContext)

	txn_mgr.Commit(txn)

	testingpkg.Assert(t, types.NewInteger(77).CompareEquals(results[0].GetValue(outSchema, 0)), "value should be 777")
}

func TestAbortWIthDeleteUpdate(t *testing.T) {
	os.Stdout.Sync()
	diskManager := disk.NewDiskManagerTest()
	defer diskManager.ShutDown()
	log_mgr := recovery.NewLogManager(&diskManager)
	bpm := buffer.NewBufferPoolManager(uint32(32), diskManager, log_mgr)
	txn_mgr := access.NewTransactionManager(log_mgr)
	txn := txn_mgr.Begin(nil)

	c := catalog.BootstrapCatalog(bpm, log_mgr, access.NewLockManager(access.REGULAR, access.PREVENTION), txn)

	columnA := column.NewColumn("a", types.Integer, false)
	columnB := column.NewColumn("b", types.Varchar, false)
	schema_ := schema.NewSchema([]*column.Column{columnA, columnB})

	tableMetadata := c.CreateTable("test_1", schema_, txn)

	row1 := make([]types.Value, 0)
	row1 = append(row1, types.NewInteger(20))
	row1 = append(row1, types.NewVarchar("hoge"))

	row2 := make([]types.Value, 0)
	row2 = append(row2, types.NewInteger(99))
	row2 = append(row2, types.NewVarchar("foo"))

	row3 := make([]types.Value, 0)
	row3 = append(row3, types.NewInteger(777))
	row3 = append(row3, types.NewVarchar("bar"))

	rows := make([][]types.Value, 0)
	rows = append(rows, row1)
	rows = append(rows, row2)
	rows = append(rows, row3)

	insertPlanNode := plans.NewInsertPlanNode(rows, tableMetadata.OID())

	executionEngine := &ExecutionEngine{}
	executorContext := NewExecutorContext(c, bpm, txn)
	executionEngine.Execute(insertPlanNode, executorContext)

	txn_mgr.Commit(txn)

	fmt.Println("update and delete rows...")
	txn = txn_mgr.Begin(nil)
	executorContext.SetTransaction(txn)

	// update
	row1 = make([]types.Value, 0)
	row1 = append(row1, types.NewInteger(99))
	row1 = append(row1, types.NewVarchar("updated"))

	pred := Predicate{"b", expression.Equal, "foo"}
	tmpColVal := new(expression.ColumnValue)
	tmpColVal.SetTupleIndex(0)
	tmpColVal.SetColIndex(tableMetadata.Schema().GetColIndex(pred.LeftColumn))
	expression_ := expression.NewComparison(*tmpColVal, expression.NewConstantValue(GetValue(pred.RightColumn), GetValueType(pred.RightColumn)), pred.Operator, types.Boolean)

	updatePlanNode := plans.NewUpdatePlanNode(row1, &expression_, tableMetadata.OID())
	executionEngine.Execute(updatePlanNode, executorContext)

	// delete
	pred = Predicate{"b", expression.Equal, "bar"}
	tmpColVal = new(expression.ColumnValue)
	tmpColVal.SetTupleIndex(0)
	tmpColVal.SetColIndex(tableMetadata.Schema().GetColIndex(pred.LeftColumn))
	expression_ = expression.NewComparison(*tmpColVal, expression.NewConstantValue(GetValue(pred.RightColumn), GetValueType(pred.RightColumn)), pred.Operator, types.Boolean)

	deletePlanNode := plans.NewDeletePlanNode(&expression_, tableMetadata.OID())
	executionEngine.Execute(deletePlanNode, executorContext)

	common.EnableLogging = false

	fmt.Println("select and check value before Abort...")

	// check updated row
	outColumnB := column.NewColumn("b", types.Varchar, false)
	outSchema := schema.NewSchema([]*column.Column{outColumnB})

	pred = Predicate{"a", expression.Equal, 99}
	tmpColVal = new(expression.ColumnValue)
	tmpColVal.SetTupleIndex(0)
	tmpColVal.SetColIndex(tableMetadata.Schema().GetColIndex(pred.LeftColumn))
	expression_ = expression.NewComparison(*tmpColVal, expression.NewConstantValue(GetValue(pred.RightColumn), GetValueType(pred.RightColumn)), pred.Operator, types.Boolean)

	seqPlan := plans.NewSeqScanPlanNode(outSchema, &expression_, tableMetadata.OID())
	results := executionEngine.Execute(seqPlan, executorContext)

	testingpkg.Assert(t, types.NewVarchar("updated").CompareEquals(results[0].GetValue(outSchema, 0)), "value should be 'updated'")

	// check deleted row
	outColumnB = column.NewColumn("b", types.Integer, false)
	outSchema = schema.NewSchema([]*column.Column{outColumnB})

	pred = Predicate{"b", expression.Equal, "bar"}
	tmpColVal = new(expression.ColumnValue)
	tmpColVal.SetTupleIndex(0)
	tmpColVal.SetColIndex(tableMetadata.Schema().GetColIndex(pred.LeftColumn))
	expression_ = expression.NewComparison(*tmpColVal, expression.NewConstantValue(GetValue(pred.RightColumn), GetValueType(pred.RightColumn)), pred.Operator, types.Boolean)

	seqPlan = plans.NewSeqScanPlanNode(outSchema, &expression_, tableMetadata.OID())
	results = executionEngine.Execute(seqPlan, executorContext)

	testingpkg.Assert(t, len(results) == 0, "")

	txn_mgr.Abort(txn)

	fmt.Println("select and check value after Abort...")

	txn = txn_mgr.Begin(nil)
	executorContext.SetTransaction(txn)

	// check updated row
	outColumnB = column.NewColumn("b", types.Varchar, false)
	outSchema = schema.NewSchema([]*column.Column{outColumnB})

	pred = Predicate{"a", expression.Equal, 99}
	tmpColVal = new(expression.ColumnValue)
	tmpColVal.SetTupleIndex(0)
	tmpColVal.SetColIndex(tableMetadata.Schema().GetColIndex(pred.LeftColumn))
	expression_ = expression.NewComparison(*tmpColVal, expression.NewConstantValue(GetValue(pred.RightColumn), GetValueType(pred.RightColumn)), pred.Operator, types.Boolean)

	seqPlan = plans.NewSeqScanPlanNode(outSchema, &expression_, tableMetadata.OID())
	results = executionEngine.Execute(seqPlan, executorContext)

	testingpkg.Assert(t, types.NewVarchar("foo").CompareEquals(results[0].GetValue(outSchema, 0)), "value should be 'foo'")

	// check deleted row
	outColumnB = column.NewColumn("b", types.Integer, false)
	outSchema = schema.NewSchema([]*column.Column{outColumnB})

	pred = Predicate{"b", expression.Equal, "bar"}
	tmpColVal = new(expression.ColumnValue)
	tmpColVal.SetTupleIndex(0)
	tmpColVal.SetColIndex(tableMetadata.Schema().GetColIndex(pred.LeftColumn))
	expression_ = expression.NewComparison(*tmpColVal, expression.NewConstantValue(GetValue(pred.RightColumn), GetValueType(pred.RightColumn)), pred.Operator, types.Boolean)

	seqPlan = plans.NewSeqScanPlanNode(outSchema, &expression_, tableMetadata.OID())
	results = executionEngine.Execute(seqPlan, executorContext)

	testingpkg.Assert(t, len(results) == 1, "")
}

type ColumnInsertMeta struct {
	/**
	 * Name of the column
	 */
	name_ string
	/**
	 * Type of the column
	 */
	type_ types.TypeID
	/**
	 * Whether the column is nullable
	 */
	nullable_ bool
	/**
	 * Distribution of values
	 */
	dist_ int32
	/**
	 * Min value of the column
	 */
	min_ int32
	/**
	 * Max value of the column
	 */
	max_ int32
	/**
	 * Counter to generate serial data
	 */
	serial_counter_ int32
}

type TableInsertMeta struct {
	/**
	 * Name of the table
	 */
	name_ string
	/**
	 * Number of rows
	 */
	num_rows_ uint32
	/**
	 * Columns
	 */
	col_meta_ []*ColumnInsertMeta
}

const DistSerial int32 = 0
const DistUniform int32 = 1

func GenNumericValues(col_meta ColumnInsertMeta, count uint32) []types.Value {
	var values []types.Value
	if col_meta.dist_ == DistSerial {
		for i := 0; i < int(count); i++ {
			values = append(values, types.NewInteger(col_meta.serial_counter_))
			col_meta.serial_counter_ += 1
		}
		return values
	}

	seed := time.Now().UnixNano()
	rand.Seed(seed)
	for i := 0; i < int(count); i++ {
		values = append(values, types.NewInteger(rand.Int31n(col_meta.max_)))
	}
	return values
}

func MakeValues(col_meta *ColumnInsertMeta, count uint32) []types.Value {
	//var values []types.Value
	switch col_meta.type_ {
	case types.Integer:
		return GenNumericValues(*col_meta, count)
	default:
		panic("Not yet implemented")
	}
}

func FillTable(info *catalog.TableMetadata, table_meta *TableInsertMeta, txn *access.Transaction) {
	var num_inserted uint32 = 0
	var batch_size uint32 = 128
	for num_inserted < table_meta.num_rows_ {
		var values []types.Value
		var num_values uint32 = uint32(math.Min(float64(batch_size), float64(table_meta.num_rows_-num_inserted)))
		for _, col_meta := range table_meta.col_meta_ {
			values = append(values, MakeValues(col_meta, num_values)...)
		}
		for i := 0; i < int(num_values); i++ {
			var entry []types.Value
			//entry.reserve(values.size())
			entry = append(entry, values...)
			//var rid page.RID
			info.Table().InsertTuple(tuple.NewTupleFromSchema(entry, info.Schema()), txn)
			//BUSTUB_ASSERT(inserted, "Sequential insertion cannot fail")
			num_inserted++
		}
		// exec_ctx_.GetBufferPoolManager().FlushAllPages()
	}
	//LOG_INFO("Wrote %d tuples to table %s.", num_inserted, table_meta.name_)
}

func MakeColumnValueExpression(schema_ *schema.Schema, tuple_idx uint32,
	col_name string) expression.Expression {
	col_idx := schema_.GetColIndex(col_name)
	col_type := schema_.GetColumn(col_idx).GetType()
	col_val := expression.NewColumnValue(tuple_idx, col_idx, col_type)
	return col_val
	//allocated_exprs = append(allocated_exprs, casted_col_val)
	//return allocated_exprs_.back().get()
}

func MakeComparisonExpression(lhs expression.Expression, rhs expression.Expression,
	comp_type expression.ComparisonType) *expression.Expression {
	//allocated_exprs_.emplace_back(std::make_unique<ComparisonExpression>(lhs, rhs, comp_type));
	//return allocated_exprs_.back().get();
	//casted_lhs := (*expression.ColumnValue)(unsafe.Pointer(lhs))
	casted_lhs := lhs.(*expression.ColumnValue)

	ret_exp := expression.NewComparison(*casted_lhs, rhs, comp_type, types.Boolean)
	return &ret_exp
}

func TestSimpleHashJoin(t *testing.T) {
	diskManager := disk.NewDiskManagerTest()
	defer diskManager.ShutDown()
	log_mgr := recovery.NewLogManager(&diskManager)
	bpm := buffer.NewBufferPoolManager(uint32(32), diskManager, log_mgr)
	txn_mgr := access.NewTransactionManager(log_mgr)
	txn := txn_mgr.Begin(nil)
	c := catalog.BootstrapCatalog(bpm, log_mgr, access.NewLockManager(access.REGULAR, access.PREVENTION), txn)
	executorContext := NewExecutorContext(c, bpm, txn)

	columnA := column.NewColumn("colA", types.Integer, false)
	columnB := column.NewColumn("colB", types.Integer, false)
	columnC := column.NewColumn("colC", types.Integer, false)
	columnD := column.NewColumn("colD", types.Integer, false)
	schema_ := schema.NewSchema([]*column.Column{columnA, columnB, columnC, columnD})
	tableMetadata1 := c.CreateTable("test_1", schema_, txn)

	column1 := column.NewColumn("col1", types.Integer, false)
	column2 := column.NewColumn("col2", types.Integer, false)
	column3 := column.NewColumn("col3", types.Integer, false)
	column4 := column.NewColumn("col3", types.Integer, false)
	schema_ = schema.NewSchema([]*column.Column{column1, column2, column3, column4})
	tableMetadata2 := c.CreateTable("test_2", schema_, txn)

	tableMeta1 := &TableInsertMeta{"test_1",
		100,
		[]*ColumnInsertMeta{
			{"colA", types.Integer, false, DistSerial, 0, 0, 0},
			{"colB", types.Integer, false, DistUniform, 0, 9, 0},
			{"colC", types.Integer, false, DistUniform, 0, 9999, 0},
			{"colD", types.Integer, false, DistUniform, 0, 99999, 0},
		}}
	tableMeta2 := &TableInsertMeta{"test_2",
		1000,
		[]*ColumnInsertMeta{
			{"col1", types.Integer, false, DistSerial, 0, 0, 0},
			{"col2", types.Integer, false, DistUniform, 0, 9, 9},
			{"col3", types.Integer, false, DistUniform, 0, 9999, 1024},
			{"col4", types.Integer, false, DistUniform, 0, 99999, 2048},
		}}
	FillTable(tableMetadata1, tableMeta1, txn)
	FillTable(tableMetadata2, tableMeta2, txn)

	txn_mgr.Commit(txn)

	var scan_plan1 plans.Plan
	var out_schema1 *schema.Schema
	{
		table_info := executorContext.GetCatalog().GetTableByName("test_1")
		//&schema := table_info.schema_
		colA := column.NewColumn("colA", types.Varchar, false)
		colB := column.NewColumn("colB", types.Varchar, false)
		out_schema1 = schema.NewSchema([]*column.Column{colA, colB})
		scan_plan1 = plans.NewSeqScanPlanNode(out_schema1, nil, table_info.OID())
	}
	var scan_plan2 plans.Plan
	var out_schema2 *schema.Schema
	{
		table_info := executorContext.GetCatalog().GetTableByName("test_2")
		//schema := table_info.schema_
		col1 := column.NewColumn("col1", types.Integer, false)
		col2 := column.NewColumn("col2", types.Integer, false)
		out_schema2 = schema.NewSchema([]*column.Column{col1, col2})
		scan_plan2 = plans.NewSeqScanPlanNode(out_schema2, nil, table_info.OID())
	}
	var join_plan *plans.HashJoinPlanNode
	var out_final *schema.Schema
	{
		// colA and colB have a tuple index of 0 because they are the left side of the join
		//var allocated_exprs []*expression.ColumnValue
		colA := MakeColumnValueExpression(out_schema1, 0, "colA")
		colA_c := column.NewColumn("colA", types.Integer, false)
		colA_c.SetIsLeft(true)
		// colB := MakeColumnValueExpression(allocated_exprs, *out_schema1, 0, "colB")
		colB_c := column.NewColumn("colB", types.Integer, false)
		colA_c.SetIsLeft(true)
		// col1 and col2 have a tuple index of 1 because they are the right side of the join
		col1 := MakeColumnValueExpression(out_schema2, 1, "col1")
		col1_c := column.NewColumn("col1", types.Integer, false)
		col1_c.SetIsLeft(false)
		// col2 := MakeColumnValueExpression(allocated_exprs, *out_schema2, 1, "col2")
		col2_c := column.NewColumn("col2", types.Integer, false)
		col1_c.SetIsLeft(false)
		var left_keys []expression.Expression
		left_keys = append(left_keys, colA)
		var right_keys []expression.Expression
		right_keys = append(right_keys, col1)
		predicate := MakeComparisonExpression(colA, col1, expression.Equal)
		out_final = schema.NewSchema([]*column.Column{colA_c, colB_c, col1_c, col2_c})
		plans_ := []plans.Plan{scan_plan1, scan_plan2}
		join_plan = plans.NewHashJoinPlanNode(out_final, plans_, *predicate,
			left_keys, right_keys)
	}

	executionEngine := &ExecutionEngine{}
	left_executor := executionEngine.CreateExecutor(join_plan.GetLeftPlan(), executorContext)
	right_executor := executionEngine.CreateExecutor(join_plan.GetRightPlan(), executorContext)
	hashJoinExecutor := NewHashJoinExecutor(executorContext, join_plan, left_executor, right_executor)
	//results := executionEngine.Execute(join_plan, executorContext)
	results := executionEngine.ExecuteWithExecutor(hashJoinExecutor)

	// executor := ExecutorFactory::CreateExecutor(GetExecutorContext(), join_plan.get())
	// executor.Init()
	// Tuple tuple
	// uint32_t num_tuples = 0
	// std::cout << "ColA, ColB, Col1, Col2" << std::endl
	// while (executor.Next(&tuple)) {
	//   std::cout << tuple.GetValue(out_final, out_schema1.GetColIdx("colA")).GetAs<int32_t>() << ", "
	// 			<< tuple.GetValue(out_final, out_schema1.GetColIdx("colB")).GetAs<int32_t>() << ", "
	// 			<< tuple.GetValue(out_final, out_schema2.GetColIdx("col1")).GetAs<int16_t>() << ", "
	// 			<< tuple.GetValue(out_final, out_schema2.GetColIdx("col2")).GetAs<int32_t>() << std::endl

	//   num_tuples++
	// }
	//ASSERT_EQ(len(result), 100)
	num_tuples := len(results)
	testingpkg.Assert(t, num_tuples == 100, "")
}
