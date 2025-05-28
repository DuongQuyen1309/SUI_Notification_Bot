package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/DuongQuyen1309/suibot/internal/config"
	"github.com/DuongQuyen1309/suibot/internal/datastore"
	"github.com/block-vision/sui-go-sdk/constant"
	"github.com/block-vision/sui-go-sdk/models"
	"github.com/block-vision/sui-go-sdk/sui"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func SUITeleNoti(ctx context.Context) error {
	var cli = sui.NewSuiClient(constant.SuiMainnetEndpoint)
	config, err := config.LoadCofig()
	if err != nil {
		fmt.Println("Error load config", err)
		return err
	}
	bot, err := CreateBot()
	if err != nil {
		fmt.Println("Error create bot", err)
		return err
	}
	var coinsSymbol = make(map[string]string)
	for _, token := range config.Wallet.Token {
		coinsSymbol[token.Address] = token.Symbol
	}
	var currentCurson *string
	for {
		req := models.SuiXQueryTransactionBlocksRequest{
			SuiTransactionBlockResponseQuery: models.SuiTransactionBlockResponseQuery{
				TransactionFilter: models.TransactionFilter{
					"ToAddress": config.Wallet.AddressId, // nó sẽ lọc hết cả fromAddress
				},
				Options: models.SuiTransactionBlockOptions{
					ShowInput:          true,
					ShowEffects:        true,
					ShowBalanceChanges: true,
				},
			},
			Cursor:          currentCurson,
			Limit:           10,
			DescendingOrder: true,
		}

		resp, err := cli.SuiXQueryTransactionBlocks(ctx, req)
		if err != nil {
			fmt.Println(err)
			return err
		}
		if err := ProcessTransactionBlock(resp, coinsSymbol, config, bot, ctx); err != nil {
			return err
		}
		currentCurson = &resp.NextCursor
	}
}

func ProcessTransactionBlock(resp models.SuiXQueryTransactionBlocksResponse, coinsSymbol map[string]string, config *config.Config, bot *tgbotapi.BotAPI, ctx context.Context) error {
	for _, tx := range resp.Data {
		digest := tx.Digest
		stringTimestamp, err := strconv.Atoi(tx.TimestampMs)
		if err != nil {
			fmt.Println("Error convert timestamp", err)
			return err
		}
		timestamp := time.UnixMilli(int64(stringTimestamp))
		for _, change := range tx.BalanceChanges {
			intAmount, err := strconv.Atoi(change.Amount)
			if err != nil {
				fmt.Println("Error convert amount", err)
				return err
			}
			var coinType string
			coinType, ok := coinsSymbol[change.CoinType]
			if !ok {
				coinType = change.CoinType
			}
			var addressOwner AddressOwner
			if err := json.Unmarshal(change.Owner, &addressOwner); err != nil {
				fmt.Println("Error unmarshal", err)
				return err
			}
			walletAddress := addressOwner.AddressOwner
			if walletAddress == config.Wallet.AddressId {
				var amount float64
				switch coinType {
				case "sui":
					amount = float64(intAmount) / float64(1000000000)
				case "hasui":
					amount = float64(intAmount) / float64(1000000000)
				case "isui":
					amount = float64(intAmount) / float64(1000000000)
				default:
					amount = float64(intAmount) / float64(1000000)
				}
				if err := SendNotification(bot, walletAddress, amount, coinType, timestamp); err != nil {
					fmt.Println("Error send notification", err)
					return err
				}
				if err := datastore.InsertDB(walletAddress, amount, change.Amount, digest, coinType, timestamp, ctx); err != nil {
					fmt.Println("Error insert db", err)
					return err
				}
			}
		}
	}
	return nil
}

type AddressOwner struct {
	AddressOwner string `json:"AddressOwner"`
}

func CreateBot() (*tgbotapi.BotAPI, error) {
	bot, err := tgbotapi.NewBotAPI("8062110103:AAF2gNUOTNJJ59ZccmtgxqfDplLbs3JQVaI")
	if err != nil {
		return nil, err
	}
	return bot, nil
}

func SendNotification(bot *tgbotapi.BotAPI, wallet string, amount float64, coinType string, timestamp time.Time) error {
	var msg tgbotapi.MessageConfig
	if amount > 0 {
		msg = tgbotapi.NewMessage(7734814066, fmt.Sprintf("TK: %s\nStatus: +%v %s\nAt :%v", wallet, amount, coinType, timestamp))
	}
	if amount < 0 {
		msg = tgbotapi.NewMessage(7734814066, fmt.Sprintf("TK: %s\nStatus: %v %s\nAt :%v", wallet, amount, coinType, timestamp))
	}
	_, err := bot.Send(msg)
	if err != nil {
		fmt.Println("Error sending message", err)
		return err
	}
	return nil
}
