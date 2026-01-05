
package services

import (
	"context"
	"crypto/ecdsa"
	"log"
	"math/big"
	"os"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	client       *ethclient.Client
	privateKey   *ecdsa.PrivateKey
	publicKey    *ecdsa.PublicKey
	address      common.Address
	nonceMutex   sync.Mutex
	currentNonce *uint64
)

func InitWallet() {
	var err error

	// Connect to RPC
	rpcURL := os.Getenv("RPC_URL")
	if rpcURL == "" {
		log.Fatal("RPC_URL not set")
	}

	client, err = ethclient.Dial(rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to Ethereum client: %v", err)
	}

	// Load private key
	privateKeyHex := os.Getenv("FAUCET_PRIVATE_KEY")
	if privateKeyHex == "" {
		log.Fatal("FAUCET_PRIVATE_KEY not set")
	}

	// Remove 0x prefix if present
	if len(privateKeyHex) > 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}

	privateKey, err = crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		log.Fatalf("Invalid private key: %v", err)
	}

	publicKey = privateKey.Public().(*ecdsa.PublicKey)
	address = crypto.PubkeyToAddress(*publicKey)

	log.Printf("✅ Faucet wallet initialized: %s", address.Hex())

	// Check balance
	balance, err := GetFaucetBalance()
	if err == nil {
		log.Printf("💰 Current balance: %s ETH", balance)
	}
}

func GetWalletAddress() string {
	return address.Hex()
}

func GetFaucetBalance() (string, error) {
	ctx := context.Background()
	balance, err := client.BalanceAt(ctx, address, nil)
	if err != nil {
		return "", err
	}

	// Convert to ETH
	balanceFloat := new(big.Float).SetInt(balance)
	ethValue := new(big.Float).Quo(balanceFloat, big.NewFloat(1e18))

	return ethValue.Text('f', 6), nil
}

func GetNextNonce() (uint64, error) {
	nonceMutex.Lock()
	defer nonceMutex.Unlock()

	ctx := context.Background()

	if currentNonce == nil {
		nonce, err := client.PendingNonceAt(ctx, address)
		if err != nil {
			return 0, err
		}
		currentNonce = &nonce
	}

	nonce := *currentNonce
	*currentNonce++

	return nonce, nil
}

func SendTransaction(to common.Address, amount *big.Int) (string, error) {
	ctx := context.Background()

	nonce, err := GetNextNonce()
	if err != nil {
		return "", err
	}

	gasLimit := uint64(21000)

	// Get gas price
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return "", err
	}

	// Get chain ID
	chainID, err := client.NetworkID(ctx)
	if err != nil {
		return "", err
	}

	tx := types.NewTransaction(nonce, to, amount, gasLimit, gasPrice, nil)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return "", err
	}

	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		// Rollback nonce on failure
		nonceMutex.Lock()
		if currentNonce != nil && *currentNonce > 0 {
			*currentNonce--
		}
		nonceMutex.Unlock()
		return "", err
	}

	return signedTx.Hash().Hex(), nil
}

func GetERC20Balance(tokenAddress common.Address) (*big.Int, error) {
	ctx := context.Background()

	// balanceOf(address) function signature
	balanceOfSignature := []byte("balanceOf(address)")
	hash := crypto.Keccak256Hash(balanceOfSignature)
	methodID := hash[:4]

	// Pad address to 32 bytes
	paddedAddress := common.LeftPadBytes(address.Bytes(), 32)

	// Combine method ID and padded address
	data := append(methodID, paddedAddress...)

	// Call contract
	msg := ethereum.CallMsg{
		To:   &tokenAddress,
		Data: data,
	}

	result, err := client.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, err
	}

	balance := new(big.Int).SetBytes(result)
	return balance, nil
}
