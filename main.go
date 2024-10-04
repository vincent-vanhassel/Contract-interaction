package main

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"

	simpleStorage "tx/interface"

	"github.com/AlecAivazis/survey/v2"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	networkUrl := "http://localhost:8545"
	hexPrivateKey := "8f2a55949038a9610f50fb23b5883af3b4ecb3c3bb792cbcefbd1542c692be63"

	client, err := ethclient.Dial(networkUrl)
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}

	privateKey, err := crypto.HexToECDSA(hexPrivateKey)
	if err != nil {
		log.Fatalf("Failed to load private key: %v", err)
	}
	prompt(client, privateKey)
}

func deployContract(client *ethclient.Client, privateKey *ecdsa.PrivateKey) common.Address {
	publicKey := privateKey.Public()
	publicKeyECDSA, _ := publicKey.(*ecdsa.PublicKey)
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatalf("Failed to get nonce: %v", err)
	}

	contractBin, err := os.ReadFile("SimpleStorage.bin")
	if err != nil {
		log.Fatalf("Failed to read contract bytecode: %v", err)
	}

	gasLimit := uint64(3500000)
	tx := types.NewContractCreation(nonce, big.NewInt(0), gasLimit, nil, contractBin)

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		log.Fatalf("Failed to get network ID: %v", err)
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		log.Fatalf("Failed to sign transaction: %v", err)
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Fatalf("Failed to send transaction: %v", err)
	}
	contractAddress := crypto.CreateAddress(fromAddress, nonce)

	fmt.Println("Contract deployed !! ")
	fmt.Printf("Transaction hash: %s\n", signedTx.Hash().Hex())
	fmt.Printf("Contract address: %s\n", contractAddress.Hex())

	code, err := client.CodeAt(context.Background(), contractAddress, nil)
	if err != nil {
		log.Fatalf("Failed to get contract code: %v", err)
	}

	if len(code) == 0 {
		log.Fatalf("No contract code found at the address")
	}
	return contractAddress
}

func checkTransaction(client *ethclient.Client, txHash common.Hash) {
	receipt, err := client.TransactionReceipt(context.Background(), txHash)
	if err != nil {
		log.Fatalf("Failed to get transaction receipt: %v", err)
	}

	if receipt == nil {
		log.Fatalf("Transaction is still pending")
	} else if receipt.Status != 1 {
		fmt.Printf("Gas used: %d\n", receipt.GasUsed)
		log.Fatalln("Transaction failed - status: ", receipt.Status)
	} else if receipt.Status == 1 {
		fmt.Println("Transaction succeeded")
	}
}

func set(client *ethclient.Client, privateKey *ecdsa.PrivateKey, contractAddress common.Address, value uint) {
	instance, err := simpleStorage.NewSimpleStorage(contractAddress, client)
	if err != nil {
		log.Fatalf("Failed to load contract instance: %v", err)
	}
	fmt.Println("Contract instance is loaded: ", instance)

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(1337)) // Remplace 1337 par l'ID de ta chaîne
	if err != nil {
		log.Fatalf("Failed to create authorized transactor: %v", err)
	}

	// Function `set`
	tx, err := instance.Set(auth, big.NewInt(int64(value)))
	if err != nil {
		log.Fatalf("Failed to call contract method: %v", err)
	}

	fmt.Printf("Transaction sent: %s\n", tx.Hash().Hex())
}

func get(client *ethclient.Client, contractAddress common.Address) {
	instance, err := simpleStorage.NewSimpleStorage(contractAddress, client)
	if err != nil {
		log.Fatalf("Failed to load contract instance: %v", err)
	}
	fmt.Println("Contract instance is loaded: ", instance)
	// Function `get`
	callOpts := &bind.CallOpts{}
	value, err := instance.Get(callOpts)
	if err != nil {
		log.Fatalf("Failed to retrieve stored value: %v", err)
	}

	fmt.Printf("Stored value is: %d\n", value)
}

func getContractAddressFromUser() common.Address {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Please enter the contract address: ")
	addressInput, _ := reader.ReadString('\n')
	addressInput = strings.TrimSpace(addressInput)

	if !common.IsHexAddress(addressInput) {
		log.Fatalf("Invalid contract address")
	}

	return common.HexToAddress(addressInput)
}

func getTransactionHashFromUser() common.Hash {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Please enter the transaction hash: ")
	hashInput, _ := reader.ReadString('\n')
	hashInput = strings.TrimSpace(hashInput)

	return common.HexToHash(hashInput)
}

func prompt(client *ethclient.Client, privateKey *ecdsa.PrivateKey) {
	fmt.Println("Welcome on Contract CLI\n ")

	answers := Checkboxes("What do you want to do ? (check one) ", []string{"Deploy the contract ", "Check the transaction ", "Use the SET function ", "Use the GET function ", "Exit"})

	if len(answers) != 1 {
		fmt.Println("Please select only one option")
		prompt(client, privateKey)
	}
	if answers[0] == "Deploy the contract " {
		fmt.Println("Processing ...\n ")
		deployContract(client, privateKey)
		fmt.Println("Done !\n ")
		prompt(client, privateKey)
	}
	if answers[0] == "Check the transaction " {
		fmt.Println("Processing ...\n ")
		txHash := getTransactionHashFromUser()
		checkTransaction(client, txHash)
		fmt.Println("Done !\n ")
		prompt(client, privateKey)
	}
	if answers[0] == "Use the SET function " {
		fmt.Println("Processing ...\n ")
		contractAddress := getContractAddressFromUser()
		set(client, privateKey, contractAddress, 42)
		fmt.Println("Done !\n ")
		prompt(client, privateKey)
	}
	if answers[0] == "Use the GET function " {
		fmt.Println("Processing ...\n ")
		contractAddress := getContractAddressFromUser()
		get(client, contractAddress)
		fmt.Println("Done !\n ")
		prompt(client, privateKey)
	}
	if answers[0] == "Exit" {
		fmt.Println("Bye Bye\n ")
		os.Exit(0)
	}
}

func Checkboxes(label string, opts []string) []string {
	res := []string{}
	prompt := &survey.MultiSelect{
		Message: label,
		Options: opts,
	}
	survey.AskOne(prompt, &res)

	return res
}
