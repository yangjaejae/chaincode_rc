package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
)

type SmartContract struct {

}

// ----- Wallet ----- //
type Wallet struct {
	Balance	uint64 		`json:"balance"`// Balance
	TxInfo	Transaction	`json:"txInfo`	// Transaction Information
}

// ----- Transaction Information ----- //
type Transaction struct {
	Trader	string 	`json:"trader"`	// Collaborator
	Amount 	uint64 	`json:"amount"`	// Transaction amount
	Date 	string 	`json:"date"`	// Transaction date
	TxType 	string 	`json:"txType"`	// Transaction type
					// 		0: Publish(By Admin)
					// 		1: Payment(By Sender) 		2: Payment(By Receiver)
					// 		3: Cancel Payment(By Sender) 	4: Cancel Payment(By Receiver)	
					// 		5: Remittance(By Sender) 	6: Remittance(By Receiver)
					// 		7: Cancel Remittance(By Sender) 8: Cancel Remittance(By Receiver)	
}

// ============================================================================================================================
// 	Main
// ============================================================================================================================
func main() {
	err := shim.Start(new(SmartContract))
	if err != nil {
		fmt.Printf("Error creating new Smart Contract: %s", err)
	}
}

// ============================================================================================================================
// 	Init
// ============================================================================================================================
func (s *SmartContract) Init(stub shim.ChaincodeStubInterface) peer.Response {
	return shim.Success(nil)
}

// ============================================================================================================================
// 	Invoke
//	init		:	invoke '{"Args":["init"]}'
//	init_wallet	:	invoke '{"Args":["init_wallet", "userId", "fromId", "date"]}'
//	publish		:	invoke '{"Args":["publish", "userId", "fromId", "amount", "date"]}'
//	transfer	:	invoke '{"Args":["transfer", "userId", "toId", "amount", "txType", "date"]}'
//	get_account	:	query '{"Args":["get_account", "userId"]}'
//	get_txList	:	query '{"Args":["get_txList", "userId"]}'
// ============================================================================================================================
func (s *SmartContract) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	function, args := stub.GetFunctionAndParameters()

	if function == "init" {
		return s.Init(stub)
	} else if function == "init_wallet" {
		return init_wallet(stub, args)
	} else if function == "publish" {
		return publish(stub, args)
	} else if function == "transfer" {
		return transfer(stub, args)
	} else if function == "get_account" {
		return get_account(stub, args)
	} else if function == "get_txList" {
		return get_txList(stub, args)
	}

	return shim.Error(fmt.Sprintf("Received unknown invoke function name: %s", function));
}

// ============================================================================================================================
//	init_wallet
//	- params: userId, fromId, date 
//	- return: Success(nil) / Error(strMsg)
// ============================================================================================================================
func init_wallet(stub shim.ChaincodeStubInterface, args []string) peer.Response {
        var newWallet Wallet

	if len(args) != 3 {
                return shim.Error("Incorrect number of arguments. Expecting 2")
        }

	newWallet.Balance = 0
	newWallet.TxInfo.Trader = args[1]
	newWallet.TxInfo.Amount = 0
	newWallet.TxInfo.Date = args[2]
	newWallet.TxInfo.TxType = "10"
        
	walletAsBytes, _ := json.Marshal(newWallet)
        err := stub.PutState(args[0], walletAsBytes)

        if (err != nil) {
                return shim.Error(fmt.Sprintf("Failed to create Wallet for '%s'", args[0]));
        }

        return shim.Success(nil)
}

// ============================================================================================================================
//	publish
//	- params: userId, fromId, amount, date
//	- return: Success(nil) / Error(strMsg)
// ============================================================================================================================
func publish(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	var admin_for_history Wallet
	var target Wallet
	
	if len(args) != 4 {
		return shim.Error("Incorrect number of arguments. Expecting 4")
	}

	adminWalletAsBytes, _ := stub.GetState(args[1])
	walletAsBytes, _ := stub.GetState(args[0])
	if adminWalletAsBytes == nil {
		return shim.Error(fmt.Sprintf("Could not locate Wallet '%s'", "admin"))
	}	
	if walletAsBytes == nil {
		return shim.Error(fmt.Sprintf("Could not locate Wallet '%s'", args[0]))
	}
	
	json.Unmarshal(adminWalletAsBytes, &admin_for_history)
	json.Unmarshal(walletAsBytes, &target)
	amount, _ := strconv.ParseUint(args[2], 10, 64)
	
	admin_for_history.Balance += amount
	admin_for_history.TxInfo.Trader = args[0]
	admin_for_history.TxInfo.Amount = amount
	admin_for_history.TxInfo.TxType = "0"	// 0 is publish
	admin_for_history.TxInfo.Date = args[3]

	/////////////////////////////////////////////////////////////////////////////////
	// for get publish history
	target.Balance += amount
	target.TxInfo.Trader = args[1]
	target.TxInfo.Amount = amount
	target.TxInfo.TxType = "0"	// 0 is publish
	target.TxInfo.Date = args[3]
	/////////////////////////////////////////////////////////////////////////////////

	adminWalletAsBytes, _ = json.Marshal(admin_for_history)
	fail := stub.PutState(args[1], adminWalletAsBytes)
	if (fail != nil) {
		return shim.Error("Currency issue failed.");
	}

	walletAsBytes, _ = json.Marshal(target)
	err := stub.PutState(args[0], walletAsBytes)
	if (err != nil) {
		return shim.Error("Currency issue failed.");
	}

	return shim.Success(nil)
}

// ============================================================================================================================
//	transfer
//	- params: userId, toId, amount, txType, date
//	- return: Success(txID) / Error(strMsg)
// ============================================================================================================================
func transfer(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 5 {
		return shim.Error("Incorrect number of arguments. Expecting 5")
	}

	sender := Wallet{}
	receiver := Wallet{}

	senderAsBytes, _ := stub.GetState(args[0])
	receiverAsBytes, _ := stub.GetState(args[1])
	if senderAsBytes == nil || receiverAsBytes == nil {
		return shim.Error("Failed to fetch Wallet.")
	}

	amount, _ := strconv.ParseUint(args[2], 10, 64)

	senderType, _ := strconv.Atoi(args[3])
	senderType += 1
	receiverType := strconv.Itoa(senderType)

	json.Unmarshal(senderAsBytes, &sender)
	json.Unmarshal(receiverAsBytes, &receiver)
	
	if sender.Balance < amount {
		return shim.Error(fmt.Sprintf("%s is not enough balance.", args[0]))
	}
	
	sender.Balance -= amount
	sender.TxInfo.Trader = args[1]
	sender.TxInfo.Amount = amount
	sender.TxInfo.TxType = args[3]
	sender.TxInfo.Date = args[4]

	receiver.Balance += amount
	receiver.TxInfo.Trader = args[0]
	receiver.TxInfo.Amount = amount
	receiver.TxInfo.TxType = receiverType
	receiver.TxInfo.Date = args[4]

	senderAsBytes, _ = json.Marshal(sender)
	receiverAsBytes, _ = json.Marshal(receiver)

	err := stub.PutState(args[0], senderAsBytes)
	if (err != nil) {
		return shim.Error(fmt.Sprintf("Currency transfer failed : %s", err.Error));
	}

	txid := stub.GetTxID()

	err = stub.PutState(args[1], receiverAsBytes)
	if (err != nil) {
		return shim.Error(fmt.Sprintf("Currency receipt failed.: %s", err.Error));
	}

	return shim.Success([]byte(txid))
}

// ============================================================================================================================
// 	get_account
//	- params: userId
//	- return: Success(balance) / Error(strMsg)
// ============================================================================================================================
func get_account(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	var wallet Wallet
	
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	 }
  
	 walletAsBytes, _ := stub.GetState(args[0]);
	 if walletAsBytes == nil {
		return shim.Error(fmt.Sprintf("Could not locate Wallet '%s'", args[0]))
	 }

	 json.Unmarshal(walletAsBytes, &wallet)
	 balance := fmt.Sprint(wallet.Balance)

	 return shim.Success([]byte(balance))
}

// ============================================================================================================================
// 	get_txList
//	- params: userId
//	- return: Success([]txHistory) / Error(strMsg)
// ============================================================================================================================
func get_txList(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	type get_History struct {
		TxId    string   	`json:"txId"`
		Value 	Wallet   	`json:"value"`
	 }
	 var history []get_History;
	 var wallet Wallet
  
	 if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	 }
  
	 transferId := args[0]
	 fmt.Printf("- start getHistoryForMarble: %s\n", transferId)
  
	 resultsIterator, err := stub.GetHistoryForKey(transferId)
	 if err != nil {
		return shim.Error(err.Error())
	 }
	 defer resultsIterator.Close()
  
	 for resultsIterator.HasNext() {
		historyData, err := resultsIterator.Next()
		if err != nil {
		   return shim.Error(err.Error())
		}
  
		var tx get_History
		tx.TxId = historyData.TxId                     
		json.Unmarshal(historyData.Value, &wallet)    
		if historyData.Value == nil {                 
		   var emptyWalletHistory Wallet
		   tx.Value = emptyWalletHistory                
		} else {
		   json.Unmarshal(historyData.Value, &wallet) 
		   tx.Value = wallet                      
		}
		history = append(history, tx)   
	 }
	 
	 fmt.Printf("- getHistoryForMarble returning:\n%s", history)
  
	 historyAsBytes, _ := json.Marshal(history)     //convert to array of bytes
	 return shim.Success(historyAsBytes)  
}
