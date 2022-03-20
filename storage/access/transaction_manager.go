//package concurrency
//package transaction
package access

import (
	"github.com/ryogrid/SamehadaDB/common"
	"github.com/ryogrid/SamehadaDB/recovery"
	"github.com/ryogrid/SamehadaDB/types"
)

/**
 * TransactionManager keeps track of all the transactions running in the system.
 */
type TransactionManager struct {
	// TODO: (SDB) must ensure atomicity
	next_txn_id types.TxnID
	// lock_manager *LockManager //__attribute__((__unused__))
	log_manager *recovery.LogManager // __attribute__((__unused__))
	// /** The global transaction latch is used for checkpointing. */
	global_txn_latch common.ReaderWriterLatch
}

var txn_map map[types.TxnID]*Transaction = make(map[types.TxnID]*Transaction)

func NewTransactionManager(log_manager *recovery.LogManager) *TransactionManager {
	return &TransactionManager{0, log_manager, common.NewRWLatch()}
}

func (transaction_manager *TransactionManager) Begin(txn *Transaction) *Transaction {
	// Acquire the global transaction latch in shared mode.
	transaction_manager.global_txn_latch.RLock()
	var txn_ret *Transaction = txn

	if txn_ret == nil {
		transaction_manager.next_txn_id += 1
		txn_ret = NewTransaction(transaction_manager.next_txn_id)
		// fmt.Printf("new transactin ID: %d\n", transaction_manager.next_txn_id)
	}

	if common.EnableLogging {
		log_record := recovery.NewLogRecordTxn(txn_ret.GetTransactionId(), txn_ret.GetPrevLSN(), recovery.BEGIN)
		lsn := transaction_manager.log_manager.AppendLogRecord(log_record)
		txn_ret.SetPrevLSN(lsn)
	}

	txn_map[txn_ret.GetTransactionId()] = txn_ret
	return txn_ret
}

func (transaction_manager *TransactionManager) Commit(txn *Transaction) {
	txn.SetState(COMMITTED)

	// Perform all deletes before we commit.
	write_set := txn.GetWriteSet()
	for len(write_set) != 0 {
		item := write_set[len(write_set)-1]
		table := item.table
		rid := item.rid
		if item.wtype == DELETE {
			// Note that this also releases the lock when holding the page latch.
			pageID := rid.GetPageId()
			tpage := CastPageAsTablePage(table.bpm.FetchPage(pageID))
			tpage.ApplyDelete(item.rid, txn, transaction_manager.log_manager)
		}
		write_set = write_set[:len(write_set)-1]
	}
	txn.SetWriteSet(write_set)

	if common.EnableLogging {
		log_record := recovery.NewLogRecordTxn(txn.GetTransactionId(), txn.GetPrevLSN(), recovery.COMMIT)
		lsn := transaction_manager.log_manager.AppendLogRecord(log_record)
		txn.SetPrevLSN(lsn)
		transaction_manager.log_manager.Flush()
	}

	// Release all the locks.
	transaction_manager.releaseLocks(txn)
	// Release the global transaction latch.
	transaction_manager.global_txn_latch.RUnlock()
}

func (transaction_manager *TransactionManager) Abort(txn *Transaction) {
	txn.SetState(ABORTED)

	// Rollback before releasing the access.
	write_set := txn.GetWriteSet()
	for len(write_set) != 0 {
		item := write_set[len(write_set)-1]
		table := item.table
		// TODO: (SDB) not ported yet (inside of if block)
		if item.wtype == DELETE {
			//table.RollbackDelete(item.rid_, txn)
		} else if item.wtype == INSERT {
			// Note that this also releases the lock when holding the page latch.
			//table.ApplyDelete(item.rid, txn)
			rid := item.rid
			// Note that this also releases the lock when holding the page latch.
			pageID := rid.GetPageId()
			tpage := CastPageAsTablePage(table.bpm.FetchPage(pageID))
			tpage.ApplyDelete(item.rid, txn, transaction_manager.log_manager)
		} else if item.wtype == UPDATE {
			//table.UpdateTuple(item.tuple, item.rid_, txn)
		}
		write_set = write_set[:len(write_set)-1]
	}
	txn.SetWriteSet(write_set)

	if common.EnableLogging {
		log_record := recovery.NewLogRecordTxn(txn.GetTransactionId(), txn.GetPrevLSN(), recovery.ABORT)
		lsn := transaction_manager.log_manager.AppendLogRecord(log_record)
		txn.SetPrevLSN(lsn)
	}

	// Release all the locks.
	transaction_manager.releaseLocks(txn)
	// Release the global transaction latch.
	transaction_manager.global_txn_latch.RUnlock()
}

func (transaction_manager *TransactionManager) BlockAllTransactions() {
	transaction_manager.global_txn_latch.WLock()
}

func (transaction_manager *TransactionManager) ResumeTransactions() {
	transaction_manager.global_txn_latch.WUnlock()
}

func (transaction_manager *TransactionManager) releaseLocks(txn *Transaction) {
	// TODO: (SDB) not ported yet
	/*
	   	var lock_set : unordered_set<RID>
	       for (item : *txn.GetExclusiveLockSet()) {
	         lock_set.emplace(item)
	       }
	       for (item : *txn.GetSharedLockSet()) {
	         lock_set.emplace(item)
	       }
	       for (locked_rid : lock_set) {
	         lock_manager.Unlock(txn, locked_rid)
	       }
	*/
}