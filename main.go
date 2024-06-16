package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

type GetBlockCountRequest struct {
	Jsonrpc string `json:"jsonrpc"`
	Id      string `json:"id"`
	Method  string `json:"method"`
}

type GetBlockCountResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Id      string `json:"id"`
	Result  struct {
		Count uint64 `json:"count"`
	} `json:"result"`
	Error interface{} `json:"error"`
}

// request struct
type CoinbaseTxSumRequest struct {
	Jsonrpc string `json:"jsonrpc"`
	Id      string `json:"id"`
	Method  string `json:"method"`
	Params  struct {
		Height1 uint64 `json:"height"`
		Height2 uint64 `json:"count"`
	} `json:"params"`
}

// response struct
type CoinbaseTxSumResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Id      string `json:"id"`
	Result  struct {
		EmissionAmount uint64 `json:"emission_amount"`
		FeeAmount      uint64 `json:"fee_amount"`
	} `json:"result"`
	Error interface{} `json:"error"`
}

var (
	cache          CoinbaseTxSumResponse
	cacheMutex     sync.Mutex
	lastUpdateTime time.Time
)

func getBlockCount() (uint64, error) {
	reqBody := GetBlockCountRequest{
		Jsonrpc: "2.0",
		Id:      "0",
		Method:  "get_block_count",
	}

	jsonReq, err := json.Marshal(reqBody)
	if err != nil {
		return 0, err
	}

	rpcURL := "http://127.0.0.1:20241/json_rpc" //
	resp, err := http.Post(rpcURL, "application/json", bytes.NewBuffer(jsonReq))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var result GetBlockCountResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		return 0, err
	}

	if result.Error != nil {
		return 0, fmt.Errorf("RPC Error: %v", result.Error)
	}

	return result.Result.Count, nil
}

func getCoinbaseTxSum(height1, height2 uint64) (CoinbaseTxSumResponse, error) {
	reqBody := CoinbaseTxSumRequest{
		Jsonrpc: "2.0",
		Id:      "0",
		Method:  "get_coinbase_tx_sum",
	}
	reqBody.Params.Height1 = height1
	reqBody.Params.Height2 = height2

	jsonReq, err := json.Marshal(reqBody)
	if err != nil {
		return CoinbaseTxSumResponse{}, err
	}

	rpcURL := "http://127.0.0.1:20241/json_rpc" //
	resp, err := http.Post(rpcURL, "application/json", bytes.NewBuffer(jsonReq))
	if err != nil {
		return CoinbaseTxSumResponse{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return CoinbaseTxSumResponse{}, err
	}

	var result CoinbaseTxSumResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		return CoinbaseTxSumResponse{}, err
	}

	if result.Error != nil {
		return CoinbaseTxSumResponse{}, fmt.Errorf("RPC Error: %v", result.Error)
	}

	return result, nil
}

func updateCache() {
	for {
		// get latest height
		height2, err := getBlockCount()
		if err != nil {
			log.Printf("Error getting block count: %v", err)
			time.Sleep(5 * time.Minute)
			continue
		}

		result, err := getCoinbaseTxSum(1, height2)
		if err != nil {
			log.Printf("Error updating cache: %v", err)
		} else {
			cacheMutex.Lock()
			cache = result
			lastUpdateTime = time.Now()
			cacheMutex.Unlock()
		}
		time.Sleep(5 * time.Minute)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	if time.Since(lastUpdateTime) > 5*time.Minute {
		http.Error(w, "Cache is outdated", http.StatusInternalServerError)
		return
	}

	//w.Header().Set("Content-Type", "application/json")
	//json.NewEncoder(w).Encode(cache)

	// sum emission_amount and fee_amount
	total := (cache.Result.EmissionAmount + cache.Result.FeeAmount) / 1000000000000

	//first airdrop 500 * 5 TSK
	//twitter task + discord task 7000 TSK
	//dev team reward first month 5000 TSK
	released := uint64(2500 + 7000 + 5000)
	total = total + released

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%d", total)
}

func CirculatingSupply(w http.ResponseWriter, r *http.Request) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	if time.Since(lastUpdateTime) > 5*time.Minute {
		http.Error(w, "Cache is outdated", http.StatusInternalServerError)
		return
	}

	//w.Header().Set("Content-Type", "application/json")
	//json.NewEncoder(w).Encode(cache)

	// sum emission_amount and fee_amount
	total := float64(cache.Result.EmissionAmount+cache.Result.FeeAmount) / 1e12

	//first airdrop 500 * 5 TSK
	//twitter task + discord task 7000 TSK
	//dev team reward first month 5000 TSK
	released := float64(2500 + 7000 + 5000)
	total = total + released

	responseData := map[string]string{
		"result": fmt.Sprintf("%.12f", total),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseData)
}

func TotalSupply(w http.ResponseWriter, r *http.Request) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	if time.Since(lastUpdateTime) > 5*time.Minute {
		http.Error(w, "Cache is outdated", http.StatusInternalServerError)
		return
	}

	// sum emission_amount and fee_amount
	total := float64(cache.Result.EmissionAmount+cache.Result.FeeAmount) / 1e12

	//block 0 reward: 553402.319999999949
	initBlockReward := 553402.319999999949
	total = total + initBlockReward

	responseData := map[string]string{
		"result": fmt.Sprintf("%.12f", total),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseData)
}

func main() {

	go updateCache()

	http.HandleFunc("/circulation", handler) //for xeggex
	http.HandleFunc("/CirculatingSupply", CirculatingSupply)
	http.HandleFunc("/TotalSupply", TotalSupply)

	fmt.Println("Starting server on 127.0.0.1:8086")
	log.Fatal(http.ListenAndServe("127.0.0.1:8086", nil))
}
