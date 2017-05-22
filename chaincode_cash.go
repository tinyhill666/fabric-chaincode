package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

// 作为p2p借贷系统中基础的资金合约
type CashChaincode struct {
}

type Account struct {
	Balance float64
}

// Init 函数初始化造币，
func (t *CashChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	//var value float64 // State of event
	var err error
	_, args := stub.GetFunctionAndParameters()
	if len(args) != 0 {
		return shim.Error("Incorrect number of arguments. Expecting 0")
	}

	// Initialize the chaincode
	var account Account
	account.Balance = 1000000
	fmt.Printf("init money = %d\n", account.Balance)

	cashByte, err := json.Marshal(account)
	if err != nil {
		return shim.Error("build json failed!")
	}

	err = stub.PutState("pbc", cashByte)
	if err != nil {
		return shim.Error(err.Error())
	}

	err = stub.PutState("cb", cashByte)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}
func (t *CashChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	fmt.Println("cash Invoke")
	function, args := stub.GetFunctionAndParameters()
	if function == "invoke" {
		// Make payment of X units from A to B
		return t.invoke(stub, args)
	} else if function == "query" {
		// the old "Query" is now implemtned in invoke
		return t.query(stub, args)
	} else if function == "transferN2A" {
		// the old "Query" is now implemtned in invoke
		return t.transferN2A(stub, args)
	}

	return shim.Error("Invalid invoke function name. Expecting \"invoke\" \"delete\" \"query\"")
}

// Transaction makes payment of X units from A to B
func (t *CashChaincode) invoke(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var A, B string        // Entities
	var Aval, Bval float64 // Asset holdings
	var X float64          // Transaction value
	var err error
	var accountA, accountB Account

	if len(args) != 3 {
		return shim.Error("Incorrect number of arguments. Expecting 3")
	}

	A = args[0]
	B = args[1]

	// Get the state from the ledger
	// TODO: will be nice to have a GetAllState call to ledger
	Avalbytes, err := stub.GetState(A)
	if err != nil {
		return shim.Error("Failed to get state")
	}
	if Avalbytes == nil {
		return shim.Error("Entity not found")
	}

	error := json.Unmarshal(Avalbytes, &accountA)
	if error != nil {
		jsonResp := "{\"Error\":\"json ummarshal " + fmt.Sprint(Avalbytes) + "\"}"
		return shim.Error(jsonResp)
	}
	Aval = accountA.Balance

	Bvalbytes, err := stub.GetState(B)
	if err == nil {
		if Bvalbytes == nil {
			Bval = 0
		} else {
			error := json.Unmarshal(Bvalbytes, &accountB)
			if error != nil {
				jsonResp := "{\"Error\":\"json ummarshal " + fmt.Sprint(Avalbytes) + "\"}"
				return shim.Error(jsonResp)
			}
			Bval = accountB.Balance
		}
	} else { //账户不存在，新建账户worldstate
		Bval = 0
	}

	// Perform the execution
	X, err = strconv.ParseFloat(args[2], 64)
	if err != nil {
		return shim.Error("Invalid transaction amount, expecting a float value")
	}
	Aval = Aval - X
	if Aval < 0 {
		return shim.Error("Account:" + A + ", balance is insufficient")
	}
	Bval = Bval + X
	fmt.Printf("Aval = %d, Bval = %d\n", Aval, Bval)

	accountB.Balance = Bval
	accountA.Balance = Aval
	// Write the state back to the ledger
	AByte, err := json.Marshal(accountA)
	if err != nil {
		return shim.Error("build json failed!")
	}

	err = stub.PutState(A, AByte)
	if err != nil {
		return shim.Error(err.Error())
	}

	BByte, err := json.Marshal(accountB)
	if err != nil {
		return shim.Error("build json failed!")
	}

	err = stub.PutState(B, BByte)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

// query callback representing the query of a chaincode
func (t *CashChaincode) query(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var A string // Entities
	var err error

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting name of the person to query")
	}

	A = args[0]

	// Get the state from the ledger
	Avalbytes, err := stub.GetState(A)
	if err != nil {
		jsonResp := "{\"Error\":\"Failed to get state for " + A + "\"}"
		return shim.Error(jsonResp)
	}

	if Avalbytes == nil {
		jsonResp := "{\"Error\":\"Nil amount for " + A + "\"}"
		return shim.Error(jsonResp)
	}

	jsonResp := "{\"Name\":\"" + A + "\",\"Amount\":\"" + string(Avalbytes) + "\"}"
	fmt.Printf("Query Response:%s\n", jsonResp)
	return shim.Success(Avalbytes)
}

// Transaction makes payment of X units from A1.A2...AN to B
func (t *CashChaincode) transferN2A(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var num int         //number of N
	var receiver string //who receive money
	var totalDelta float64
	//var Aval, Bval float64 // Asset holdings
	//var X float64          // Transaction value
	var err error
	//var accountA, accountB Account

	if (len(args) % 2) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting odd number!")
	}

	num = (len(args) - 1) / 2
	receiver = args[0]
	totalDelta = 0

	count := 0
	for count < num {
		sender := args[count*2+1]
		deltaString := args[count*2+2]
		count += 1
		delta, err := strconv.ParseFloat(deltaString, 64)
		totalDelta += delta

		var senderAccount Account
		senderBytes, err := stub.GetState(sender)
		if err != nil {
			return shim.Error("Failed to get state")
		}
		if senderBytes == nil {
			return shim.Error("Entity not found")
		}

		error := json.Unmarshal(senderBytes, &senderAccount)
		if error != nil {
			jsonResp := "{\"Error\":\"json ummarshal " + fmt.Sprint(senderBytes) + "\"}"
			return shim.Error(jsonResp)
		}
		value := senderAccount.Balance
		value -= delta
		if value < 0 {
			return shim.Error("Account:" + sender + ", balance is insufficient")
		}
		senderAccount.Balance = value

		// Write the state back to the ledger
		SByte, err := json.Marshal(senderAccount)
		if err != nil {
			return shim.Error("build json failed!")
		}

		err = stub.PutState(sender, SByte)
		if err != nil {
			return shim.Error(err.Error())
		}

	}

	//receiver
	var receiverAccount Account
	var receiverValue float64
	receiverBytes, err := stub.GetState(receiver)
	if err == nil {
		if receiverBytes == nil {
			receiverValue = 0
		} else {
			error := json.Unmarshal(receiverBytes, &receiverAccount)
			if error != nil {
				jsonResp := "{\"Error\":\"json ummarshal " + fmt.Sprint(receiverBytes) + "\"}"
				return shim.Error(jsonResp)
			}
			receiverValue = receiverAccount.Balance
		}
	} else { //账户不存在，新建账户worldstate
		receiverValue = 0
	}

	receiverValue += totalDelta
	receiverAccount.Balance = receiverValue

	// Write the state back to the ledger
	RByte, err := json.Marshal(receiverAccount)
	if err != nil {
		return shim.Error("build json failed!")
	}

	err = stub.PutState(receiver, RByte)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

func main() {
	err := shim.Start(new(CashChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}
