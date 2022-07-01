package db

import (
	"context"
	"database/sql"
	"fmt"
)

// provides all functions to execute db queries and transactions
type Store struct {
	*Queries
	db *sql.DB
}

// contains the input parameters of the transfer transaction
type TransferTxParams struct {
	FromAccountID int64 `json:"from_account_id"`
	ToAccountID   int64 `json:"to_account_id"`
	Amount        int64 `json:"amount"`
}

// result of transfer transaction
type TransferTxResult struct {
	Transfer Transfer `json:"transfer"`
	FromAccount Account `json:"from_account"`
	ToAccount Account `json:"to_account"`
	FromEntry Entry `json:"from_entry"`
	ToEntry Entry `json:"to_entry"`
}

// creates a new store instance
func NewStore(db *sql.DB) *Store {
	return &Store{
		db:      db,
		Queries: New(db),
	}
}

// executes a function within a transaction
func (s *Store) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	q := New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, tx rollback: %v", err, rbErr)
		}
	}

	return tx.Commit()
}

// performs money transfer from one account to another
// creates a transfer record, adds account entries and updates accounts' balance in a single transaction
func (s *Store) TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error) {
	var result TransferTxResult

	err := s.execTx(ctx, func(q *Queries) error {
		var err error

		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams{
			Amount: arg.Amount,
			FromAccountID: arg.FromAccountID,
			ToAccountID: arg.ToAccountID,
		})
		if err != nil {
			return err
		}

		result.FromEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.FromAccountID,
			Amount: -arg.Amount,
		})
		if err != nil {
			return err
		}

		result.ToEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.ToAccountID,
			Amount: arg.Amount,
		})
		if err != nil {
			return err
		}

		if arg.FromAccountID > arg.ToAccountID {
			result.FromAccount, result.ToAccount, err = addMoney(ctx, q, arg.FromAccountID, -arg.Amount, result.Transfer.ToAccountID, arg.Amount)
		} else {
			result.ToAccount, result.FromAccount, err = addMoney(ctx, q, arg.ToAccountID, arg.Amount, result.Transfer.FromAccountID, -	arg.Amount)
		}

		return nil
	})

	return result, err
}


func addMoney(
	ctx context.Context,
	q *Queries,
	accountID1 int64,
	amount1 int64,
	accountID2 int64,
	amount2 int64,
) (account1 Account, account2 Account, err error) {
	account1, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
		ID: accountID1,
		Amount: amount1,
	})
	if err != nil {
		return
	}

	account2, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
		ID: accountID2,
		Amount: amount2,
	})
	if err != nil {
		return
	}

	return
}