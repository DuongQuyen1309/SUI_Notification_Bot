package datastore

import (
	"context"
	"fmt"
	"time"

	"github.com/DuongQuyen1309/suibot/internal/db"
	"github.com/DuongQuyen1309/suibot/internal/model"
	"github.com/uptrace/bun"
)

func CreateTransactionsTable(DB *bun.DB, ctx context.Context) error {
	_, err := DB.NewCreateTable().Model((*model.SuiTransaction)(nil)).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return err
	}
	_, err = DB.NewCreateIndex().Model((*model.SuiTransaction)(nil)).
		Index("idx_transaction_hash_address_token").
		Unique().
		Column("transaction_hash", "wallet_address", "token").
		IfNotExists().
		Exec(ctx)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}
func InsertDB(wallet string, amount float64, rawAmount string, digest string, symbol string, timestamp time.Time, ctx context.Context) error {
	_, err := db.DB.NewInsert().Model(&model.SuiTransaction{
		WalletAddress:   wallet,
		Amount:          amount,
		RawAmount:       rawAmount,
		Token:           symbol,
		TransactionHash: digest,
		CreatedAt:       timestamp,
	}).Exec(ctx)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}
