package main

import (
	"context"
	"fmt"
	"gopkg.in/yaml.v2"
	"log"
	"os"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
)

type Config struct {
	TransferConfig struct {
		RPCURL      string `yaml:"rpcURL"`
		PrivateKey  string `yaml:"privateKey"`
		RecipientPK string `yaml:"recipientPK"`
		Amount      uint64 `yaml:"amount"`
	} `yaml:"transfer_config"`
}

func main() {
	configFile, err := os.Open("config.yaml")
	if err != nil {
		log.Fatalf("failed to open config file: %v", err)
	}
	defer configFile.Close()

	var config Config
	err = yaml.NewDecoder(configFile).Decode(&config)
	if err != nil {
		log.Fatalf("failed to decode config file: %v", err)
	}

	ctx := context.Background()

	rpcClient := rpc.New(config.TransferConfig.RPCURL)

	accountFrom := solana.MustPrivateKeyFromBase58(config.TransferConfig.PrivateKey)
	accountTo := solana.MustPublicKeyFromBase58(config.TransferConfig.RecipientPK)

	if err := validateConfig(config); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	if err := transferTokens(ctx, rpcClient, accountFrom, accountTo, config.TransferConfig.Amount); err != nil {
		log.Fatalf("failed to transfer tokens: %v", err)
	}

	fmt.Println("Transaction successful")
}

func validateConfig(config Config) error {
	if config.TransferConfig.Amount <= 0 {
		return fmt.Errorf("amount must be greater than zero")
	}
	return nil
}

func transferTokens(ctx context.Context, rpcClient *rpc.Client, accountFrom solana.PrivateKey, accountTo solana.PublicKey, amount uint64) error {
	balance, err := getAccountBalance(ctx, rpcClient, accountFrom.PublicKey())
	if err != nil {
		return fmt.Errorf("failed to get account balance: %v", err)
	}

	if balance < amount {
		return fmt.Errorf("insufficient balance. Account balance: %d, Required amount: %d", balance, amount)
	}

	recentBlockhash, err := rpcClient.GetRecentBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return fmt.Errorf("failed to get recent blockhash: %v", err)
	}

	tx, err := solana.NewTransaction(
		[]solana.Instruction{
			system.NewTransferInstruction(
				amount,
				accountFrom.PublicKey(),
				accountTo,
			).Build(),
		},
		recentBlockhash.Value.Blockhash,
		solana.TransactionPayer(accountFrom.PublicKey()),
	)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %v", err)
	}

	signer := func(key solana.PublicKey) *solana.PrivateKey {
		if accountFrom.PublicKey().Equals(key) {
			return &accountFrom
		}
		return nil
	}

	_, err = tx.Sign(signer)
	if err != nil {
		return fmt.Errorf("unable to sign transaction: %v", err)
	}

	sig, err := rpcClient.SendTransaction(ctx, tx)
	if err != nil {
		return fmt.Errorf("failed to send transaction: %v", err)
	}

	fmt.Println("Transaction Signature:", sig)

	return nil
}

func getAccountBalance(ctx context.Context, rpcClient *rpc.Client, account solana.PublicKey) (uint64, error) {
	balanceResult, err := rpcClient.GetBalance(ctx, account, rpc.CommitmentFinalized)
	if err != nil {
		return 0, fmt.Errorf("failed to get account balance: %v", err)
	}
	return balanceResult.Value, nil
}
