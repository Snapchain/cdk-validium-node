package main

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xPolygon/cdk-validium-node/config"
	"github.com/0xPolygon/cdk-validium-node/etherman/smartcontracts/cdkdatacommittee"
	"github.com/0xPolygon/cdk-validium-node/log"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli/v2"
)

func setupCommittee(ctx *cli.Context) error {
	c, err := config.Load(ctx, true)
	if err != nil {
		return err
	}
	if !ctx.Bool(config.FlagYes) {
		fmt.Println("*WARNING* Are you sure you want to setup committee? [y/N]: ")
		var input string
		if _, err := fmt.Scanln(&input); err != nil {
			return err
		}
		input = strings.ToLower(input)
		if !(input == "y" || input == "yes") {
			return nil
		}
	}

	setupLog(c.Log)

	// Check if it is already registered
	etherman, err := newEtherman(*c)
	if err != nil {
		log.Fatal(err)
		return err
	}

	// load auth from keystore file
	addrKeyStorePath := ctx.String(config.FlagKeyStorePath)
	addrPassword := ctx.String(config.FlagPassword)
	authL1, _, err := etherman.LoadAuthFromKeyStore(addrKeyStorePath, addrPassword)
	if err != nil {
		log.Fatal(err)
		return err
	}

	dataCommitteeContract := c.DataAvailability.L1.DataCommitteeAddress
	fmt.Println("cdkDataCommitteeContract: ", dataCommitteeContract)

	clientL1, err := ethclient.Dial(c.DataAvailability.L1.RpcURL)
	if err != nil {
		log.Fatal(err)
		return err
	}
	dacSC, err := cdkdatacommittee.NewCdkdatacommittee(
		common.HexToAddress(dataCommitteeContract),
		clientL1,
	)
	if err != nil {
		log.Fatal(err)
		return err
	}

	const nSignatures = 1
	addrsBytes := []byte{}
	urls := []string{}

	// load dac member from keystore file
	keystoreEncrypted, err := os.ReadFile(filepath.Clean(c.DataAvailability.PrivateKey.Path))
	if err != nil {
		return err
	}
	log.Infof("decrypting dac member key from: %v", c.DataAvailability.PrivateKey.Path)
	dacMemberKey, err := keystore.DecryptKey(keystoreEncrypted, c.DataAvailability.PrivateKey.Password)
	if err != nil {
		return err
	}
	dacMemberAddress := dacMemberKey.Address
	fmt.Println("dacMemberAddress: ", dacMemberAddress)
	dacServiceURL := fmt.Sprintf("http://%s:%d", c.DataAvailability.RPC.Host, c.DataAvailability.RPC.Port)
	fmt.Println("dacServiceURL: ", dacServiceURL)
	addrsBytes = append(addrsBytes, dacMemberAddress.Bytes()...)
	urls = append(urls, dacServiceURL)

	tx, err := dacSC.SetupCommittee(authL1, big.NewInt(nSignatures), urls, addrsBytes)
	if err != nil {
		return err
	}
	const (
		mainnet = 1
		rinkeby = 4
		goerli  = 5
		local   = 1337
	)
	switch c.NetworkConfig.L1Config.L1ChainID {
	case mainnet:
		fmt.Println("Check tx status: https://etherscan.io/tx/" + tx.Hash().String())
	case rinkeby:
		fmt.Println("Check tx status: https://rinkeby.etherscan.io/tx/" + tx.Hash().String())
	case goerli:
		fmt.Println("Check tx status: https://goerli.etherscan.io/tx/" + tx.Hash().String())
	case local:
		fmt.Println("Local network. Tx Hash: " + tx.Hash().String())
	default:
		fmt.Println("Unknown network. Tx Hash: " + tx.Hash().String())
	}
	return nil
}
