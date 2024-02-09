package tools

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

func ContractState(dbPath string, addr string) {
	db := getDb(dbPath)
	stateRootNode := getStateTrees(db, 1)[0].stateRoot
	getStateForContract(db, stateRootNode, addr)
}

/*
==================================================================================================================================
*/

type stateFound struct {
	blockNumber *big.Int
	stateRoot   common.Hash
}

func getStateTrees(db ethdb.Database, stopAt int) []stateFound {
	var res []stateFound
	headerHash, _ := db.Get(headHeaderKey)
	for headerHash != nil {
		// print the header hash
		var blockHeader types.Header
		blockNb, _ := db.Get(append(headerNumberPrefix, headerHash...))
		if blockNb == nil {
			break
		}
		blockHeaderRaw, _ := db.Get(append(headerPrefix[:], append(blockNb, headerHash...)...))
		rlp.DecodeBytes(blockHeaderRaw, &blockHeader)

		stateRootNode, _ := db.Get(blockHeader.Root.Bytes())

		if len(stateRootNode) > 0 {
			res = append(res, stateFound{blockHeader.Number, blockHeader.Root})
			if stopAt > 0 && len(res) == stopAt {
				return res
			}
		}

		headerHash = blockHeader.ParentHash.Bytes()
	}

	return res
}

func getStateForContract(db ethdb.Database, stateRootNode common.Hash, addr string) {

	trieDB := trie.NewDatabase(db, nil)
	treeState, _ := trie.New(trie.StateTrieID(stateRootNode), trieDB)

	addrHash := crypto.Keccak256Hash(common.Hex2Bytes(addr))

	addrState, _ := treeState.Get(addrHash.Bytes())
	var values [][]byte
	if err := rlp.DecodeBytes(addrState, &values); err != nil {
		panic(err)
	}

	fmt.Println("values", values)

	//  scp -r blast-test:/home/ec2-user/deployment/data /Users/primusa/blast-data
	// go run -x main.go /Users/primusa/blast-data/data/geth/chaindata 4200000000000000000000000000000000000023
	// decoded value must be length 4
	// 0: nonce
	// 1: balance
	// 2: storage trie
	// 3: code hash

	// get the storage trie
	storageTrie, _ := trie.New(trie.StorageTrieID(stateRootNode, common.BytesToHash(values[2]), common.BytesToHash(values[2])), trieDB)
	storageIterator, _ := storageTrie.NodeIterator(nil)
	it := trie.NewIterator(storageIterator)
	for it.Next() {
		var value []byte
		if err := rlp.DecodeBytes(it.Value, &value); err != nil {
			panic(err)
		}
		// print out hex encoded key and value
		fmt.Printf("0x%x: 0x%x\n", it.Key, value)
	}
}

func getDb(dbPath string) ethdb.Database {
	dbType := rawdb.PreexistingDatabase(dbPath)

	// if its levelDb
	if dbType == "leveldb" {
		db, err := rawdb.NewLevelDBDatabase(dbPath, 0, 0, "", true)
		if err != nil {
			fmt.Println("Did not find leveldb at path:", dbPath)
			fmt.Println("Are you sure you are pointing to the 'chaindata' folder?")
			panic(err)
		}
		return db
	} else if dbType == "pebble" {
		db, err := rawdb.NewPebbleDBDatabase(dbPath, 0, 0, "", true, false)
		if err != nil {
			fmt.Println("Did not find pebble at path:", dbPath)
			fmt.Println("Are you sure you are pointing to the 'chaindata' folder?")
			panic(err)
		}
		return db
	}
	panic("Database type not supported")
}
