package model

import (
	"time"

	"github.com/uptrace/bun"
)

type SuiTransaction struct {
	bun.BaseModel `bun:"table:sui_transactions`

	Id              int       `bun:"id,pk,autoincrement"`
	WalletAddress   string    `bun:"wallet_address"`
	Amount          float64   `bun:"amount"`
	RawAmount       string    `bun:"raw_amount"`
	Token           string    `bun:"token"`
	TransactionHash string    `bun:"transaction_hash"`
	CreatedAt       time.Time `bun:"created_at"`
}
