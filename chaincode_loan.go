/*
Copyright IBM Corp. 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

// 作为p2p借贷系统中投资方所拥有的资产合约，表示谁欠他钱
type SimpleChaincode struct {
}

//json序列化 字段需要大写开头
type Loan struct {
	LendFrom string
	LendTo   string
	Balance  float64
	Interest float64
}

// Init 函数初始化造币，
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {

	return shim.Success(nil)
}
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	if function == "offer" {
		return t.offer(stub, args)
	} else if function == "query" {
		return t.query(stub, args)
	}

	return shim.Error("Invalid invoke function name. Expecting \"invoke\" \"delete\" \"query\"")
}

func (t *SimpleChaincode) offer(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var LendFrom string
	var LendTo string
	var Balance float64
	var ccId string
	var chainCodeToCall string
	var err error

	if len(args) != 5 {
		return shim.Error("Incorrect number of arguments. Expecting 5")
	}

	ccId = args[0]
	LendFrom = args[1]
	LendTo = args[2]
	Balance, err = strconv.ParseFloat(args[3], 64)
	chainCodeToCall = args[4]
	if err != nil {
		return shim.Error("Invalid transaction amount, expecting a float value")
	}

	var loan Loan
	loan.Balance = Balance
	loan.Interest = 0
	loan.LendFrom = LendFrom
	loan.LendTo = LendTo

	loanByte, err := json.Marshal(loan)
	if err != nil {
		return shim.Error("build json failed!")
	}

	err = stub.PutState(ccId, loanByte)
	if err != nil {
		return shim.Error(err.Error())
	}

	//调用转账合约
	f := "invoke"
	invokeArgs := util.ToChaincodeArgs(f, loan.LendFrom, loan.LendTo, strconv.FormatFloat(loan.Balance, 'f', 0, 64))
	response := stub.InvokeChaincode(chainCodeToCall, invokeArgs, "")
	if response.Status != shim.OK {
		errStr := fmt.Sprintf("Failed to invoke chaincode. Got error: %s", string(response.Payload))
		fmt.Printf(errStr)
		return shim.Error(errStr)
	}

	jsonResp := "ccid:" + ccId + ",From:" + loan.LendFrom + ",To:" + loan.LendTo + ",value:" + strconv.FormatFloat(loan.Balance, 'f', 2, 64)
	return shim.Success([]byte(jsonResp))
}

// query callback representing the query of a chaincode
func (t *SimpleChaincode) query(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var ccId string // Entities
	var err error

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting name of the person to query")
	}

	ccId = args[0]

	// Get the state from the ledger
	loanByte, err := stub.GetState(ccId)
	if err != nil {
		jsonResp := "{\"Error\":\"Failed to get state for " + ccId + "\"}"
		return shim.Error(jsonResp)
	}

	if loanByte == nil {
		jsonResp := "{\"Error\":\"Nil amount for " + ccId + "\"}"
		return shim.Error(jsonResp)
	}

	var loan Loan
	error := json.Unmarshal(loanByte, &loan)
	if error != nil {
		jsonResp := "{\"Error\":\"json ummarshal " + fmt.Sprint(loanByte) + "\"}"
		return shim.Error(jsonResp)
	}
	jsonResp := "ccid:" + ccId + ",From:" + loan.LendFrom + ",To:" + loan.LendTo + ",value:" + strconv.FormatFloat(loan.Balance, 'f', 2, 64)
	return shim.Success([]byte(jsonResp))
}

func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}
