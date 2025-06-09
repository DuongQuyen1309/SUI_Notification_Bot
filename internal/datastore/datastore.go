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
		Index("idx_transaction_hash").
		Column("transaction_hash").
		IfNotExists().
		Exec(ctx)
	if err != nil {
		fmt.Println(err)
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
	}).On("CONFLICT (transaction_hash, wallet_address, token) DO NOTHING").Exec(ctx)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

// func GetTransactionBlockByHash(hash string, ctx context.Context) (*[]model.SuiTransaction, error) {
// 	var transaction []model.SuiTransaction
// 	err := db.DB.NewSelect().Model(&transaction).Where("transaction_hash = ?", hash).Scan(ctx)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &transaction, nil
// }

// func GetbalanceChangeByHashAndWalletAndCoinType(hash string, wallet string, coinType string, ctx context.Context) (*model.SuiTransaction, error) {
// 	var transaction model.SuiTransaction
// 	err := db.DB.NewSelect().
// 		Model(&transaction).
// 		Where("transaction_hash = ?", hash).
// 		Where("wallet_address = ?", wallet).
// 		Where("token = ?", coinType).
// 		Scan(ctx)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &transaction, nil
// }
