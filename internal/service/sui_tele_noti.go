package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/DuongQuyen1309/suibot/internal/config"
	"github.com/DuongQuyen1309/suibot/internal/datastore"
	"github.com/block-vision/sui-go-sdk/constant"
	"github.com/block-vision/sui-go-sdk/models"
	"github.com/block-vision/sui-go-sdk/sui"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	coinDecimals  = make(map[string]int)
	bot           *tgbotapi.BotAPI
	wg            sync.WaitGroup
	configuration *config.Config
)

func SUITeleNoti(ctx context.Context) error {
	var client = sui.NewSuiClient(constant.SuiMainnetEndpoint)
	var err error
	configuration, err = config.LoadCofig()
	if err != nil {
		fmt.Println("Error load config", err)
		return err
	}
	bot, err = CreateBot()
	if err != nil {
		fmt.Println("Error create bot", err)
		return err
	}
	for _, token := range configuration.Wallet.Token {
		coinDecimals[token.Address] = token.Decimals
	}

	latestCheckPointNumber, err := client.SuiGetLatestCheckpointSequenceNumber(ctx)
	if err != nil {
		fmt.Println("Error getting latest checkpoint", err)
		return err
	}
	newestcheckpoint := latestCheckPointNumber
	wg.Add(2)
	go func() {
		err = FilterInRealtime(client, ctx, int(newestcheckpoint))
		if err != nil {
			fmt.Println("Error filtering realtime", err)
			return
		}
	}()
	go func() {
		err = FilterInPast(ctx, client)
		if err != nil {
			fmt.Println("Error filtering in past", err)
			return
		}
	}()
	wg.Wait()
	if err != nil {
		return err
	}
	return nil
}

func FilterInPast(ctx context.Context, client sui.ISuiAPI) error {
	var err error
	go func() {
		err = FilterTransactionReceivedInPast(ctx, client)
		if err != nil {
			return
		}
	}()
	go func() {
		err = FilterTransactionSentInPast(ctx, client)
		if err != nil {
			return
		}
	}()
	if err != nil {
		return err
	}
	return nil
}
func FilterTransactionReceivedInPast(ctx context.Context, client sui.ISuiAPI) error {
	var currentCurson *string
	for {
		req := models.SuiXQueryTransactionBlocksRequest{
			SuiTransactionBlockResponseQuery: models.SuiTransactionBlockResponseQuery{
				TransactionFilter: models.TransactionFilter{
					"ToAddress": configuration.Wallet.AddressId,
				},
				Options: models.SuiTransactionBlockOptions{
					ShowInput:          false,
					ShowEffects:        false,
					ShowBalanceChanges: true,
				},
			},
			Cursor:          currentCurson,
			Limit:           10,
			DescendingOrder: true,
		}
		resp, err := QueryTransactionBlocks(client, ctx, req)
		if err != nil {
			return err
		}
		currentCurson = &resp.NextCursor
		time.Sleep(200 * time.Millisecond)
	}
}
func QueryTransactionBlocks(client sui.ISuiAPI, ctx context.Context, req models.SuiXQueryTransactionBlocksRequest) (*models.SuiXQueryTransactionBlocksResponse, error) {
	resp, err := client.SuiXQueryTransactionBlocks(ctx, req)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	if err := ProcessTransactionBlock(resp, ctx); err != nil {
		return nil, err
	}
	return &resp, nil
}
func FilterTransactionSentInPast(ctx context.Context, client sui.ISuiAPI) error {
	var currentCurson *string
	for {
		req := models.SuiXQueryTransactionBlocksRequest{
			SuiTransactionBlockResponseQuery: models.SuiTransactionBlockResponseQuery{
				TransactionFilter: models.TransactionFilter{
					"FromAddress": configuration.Wallet.AddressId,
				},
				Options: models.SuiTransactionBlockOptions{
					ShowInput:          false,
					ShowEffects:        false,
					ShowBalanceChanges: true,
				},
			},
			Cursor:          currentCurson,
			Limit:           10,
			DescendingOrder: true,
		}
		resp, err := QueryTransactionBlocks(client, ctx, req)
		if err != nil {
			return err
		}
		currentCurson = &resp.NextCursor
		time.Sleep(200 * time.Millisecond)
	}
}
func FilterInRealtime(client sui.ISuiAPI, ctx context.Context, newestCheckpoint int) error {
	for {
		req := models.SuiGetCheckpointRequest{
			CheckpointID: strconv.Itoa(int(newestCheckpoint) + 1),
		}
		_, err := client.SuiGetCheckpoint(ctx, req)
		if err != nil {
			time.Sleep(10 * time.Second)
			continue
		}
		if err := HandleACheckpoint(strconv.Itoa(int(newestCheckpoint)), ctx, client); err != nil {
			fmt.Println("Error check point :", newestCheckpoint, err)
		}
		newestCheckpoint++
	}
}

func HandleACheckpoint(currentCheckpoint string, ctx context.Context, client sui.ISuiAPI) error {
	var currentCurson *string
	for {
		req := models.SuiXQueryTransactionBlocksRequest{
			SuiTransactionBlockResponseQuery: models.SuiTransactionBlockResponseQuery{
				TransactionFilter: models.TransactionFilter{
					"Checkpoint": currentCheckpoint,
				},
				Options: models.SuiTransactionBlockOptions{
					ShowInput:          true,
					ShowEffects:        true,
					ShowBalanceChanges: true,
				},
			},
			Cursor:          currentCurson,
			Limit:           5,
			DescendingOrder: true,
		}
		resp, err := client.SuiXQueryTransactionBlocks(ctx, req)
		if err != nil {
			fmt.Println(err)
			return err
		}
		for _, tx := range resp.Data {
			err := HandleBalanceChangeOfTransactionBlock(tx, ctx)
			if err != nil {
				return err
			}
		}
		currentCurson = &resp.NextCursor
		time.Sleep(200 * time.Millisecond)
	}
}
func HandleBalanceChangeOfTransactionBlock(tx models.SuiTransactionBlockResponse, ctx context.Context) error {
	digest := tx.Digest
	stringTimestamp, err := strconv.Atoi(tx.TimestampMs)
	if err != nil {
		fmt.Println("Error convert timestamp", err)
		return err
	}
	timestamp := time.UnixMilli(int64(stringTimestamp))
	for _, change := range tx.BalanceChanges {
		rawAmount, err := strconv.Atoi(change.Amount)
		if err != nil {
			fmt.Println("Error convert amount", err)
			return err
		}
		coinType := change.CoinType
		var addressOwner AddressOwner
		if err := json.Unmarshal(change.Owner, &addressOwner); err != nil {
			fmt.Println("Error unmarshal", err)
			return err
		}
		walletAddress := addressOwner.AddressOwner
		if walletAddress == configuration.Wallet.AddressId {
			amount := float64(rawAmount) / float64(coinDecimals[coinType])
			if err := SendNotification(walletAddress, amount, coinType, timestamp); err != nil {
				fmt.Println("Error send notification", err)
				return err
			}
			if err := datastore.InsertDB(walletAddress, amount, change.Amount, digest, coinType, timestamp, ctx); err != nil {
				fmt.Println("Error insert db", err)
				return err
			}
		}
	}
	return nil
}
func ProcessTransactionBlock(resp models.SuiXQueryTransactionBlocksResponse, ctx context.Context) error {
	for _, tx := range resp.Data {
		err := HandleBalanceChangeOfTransactionBlock(tx, ctx)
		if err != nil {
			return err
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

func SendNotification(wallet string, amount float64, coinType string, timestamp time.Time) error {
	var msg tgbotapi.MessageConfig
	if amount > 0 {
		msg = tgbotapi.NewMessage(7734814066, fmt.Sprintf("TK: %s\nBalance Change: +%v %s\nAt :%v", wallet, amount, coinType, timestamp))
	}
	if amount < 0 {
		msg = tgbotapi.NewMessage(7734814066, fmt.Sprintf("TK: %s\nBalance Change: %v %s\nAt :%v", wallet, amount, coinType, timestamp))
	}
	_, err := bot.Send(msg)
	if err != nil {
		fmt.Println("Error sending message", err)
		return err
	}
	return nil
}
