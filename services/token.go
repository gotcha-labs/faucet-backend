package services

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"strings"
)

// ERC20 Transfer ABI
const erc20TransferABI = `[{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"type":"function"}]`

// SendERC20 sends ERC20 tokens
func SendERC20(tokenAddress common.Address, to common.Address, amount *big.Int) (string, error) {
	ctx := context.Background()

	// Parse ABI
	parsedABI, err := abi.JSON(strings.NewReader(erc20TransferABI))
	if err != nil {
		return "", err
	}

	// Encode transfer function call
	data, err := parsedABI.Pack("transfer", to, amount)
	if err != nil {
		return "", err
	}

	nonce, err := GetNextNonce()
	if err != nil {
		return "", err
	}

	gasLimit := uint64(100000) // ERC20 transfer typically needs ~65k gas

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return "", err
	}

	chainID, err := client.NetworkID(ctx)
	if err != nil {
		return "", err
	}

	tx := types.NewTransaction(
		nonce,
		tokenAddress,
		big.NewInt(0), // No ETH value for ERC20 transfer
		gasLimit,
		gasPrice,
		data,
	)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return "", err
	}

	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		// Rollback nonce
		nonceMutex.Lock()
		if currentNonce != nil && *currentNonce > 0 {
			*currentNonce--
		}
		nonceMutex.Unlock()
		return "", err
	}

	return signedTx.Hash().Hex(), nil
}
