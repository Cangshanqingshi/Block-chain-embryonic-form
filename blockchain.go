package main

import (
	"github.com/boltdb/bolt"
	"log"
	"encoding/hex"
	"bytes"
	"crypto/ecdsa"
	"errors"
	"fmt"
)

const dbFile = "blockchain.db"

const blocksBucket = "blocks"

//创世块中的信息
const genesisCoinbaseData = "The Times 03/Jan/2009 Chancellor on brink of second bailout for banks"

//区块链
type Blockchain struct {
    tip []byte
    db  *bolt.DB
}

//工厂模式db
func(bc *Blockchain) Db() *bolt.DB {
	return bc.db
}
 
//把区块添加进区块链,挖矿
func (bc *Blockchain) MineBlock(transactions []*Transaction) {
	var lastHash []byte

	//在一笔交易被放入一个块之前进行验证
	for _, tx := range transactions {
		if bc.VerifyTransaction(tx) != true {
			log.Panic("ERROR: Invalid transaction")
		}
	}

	//只读的方式浏览数据库，获取当前区块链顶端区块的哈希，为加入下一区块做准备
	err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		lastHash = b.Get([]byte("l"))	//通过键"l"拿到区块链顶端区块哈希
 
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
 
	//prevBlock := bc.Blocks[len(bc.Blocks)-1]
	//求出新区块
	newBlock := NewBlock(transactions,lastHash)
	// bc.Blocks = append(bc.Blocks,newBlock)
	//把新区块加入到数据库区块链中
	err = bc.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		err := b.Put(newBlock.Hash,newBlock.Serialize())
		if err != nil {
			log.Panic(err)
		}
		err = b.Put([]byte("l"),newBlock.Hash)
		bc.tip = newBlock.Hash
 
		return nil
	})
}

//创建创世块
func NewGenesisBlock(coinbase *Transaction) *Block {
	return NewBlock([]*Transaction{coinbase},[]byte{})
}

/*新的创建区块链的函数
1.	打开一个数据库文件
2.	检查文件里面是否已经存储了一个区块链
3.	如果已经存储了一个区块链：
（1） 创建一个新的 Blockchain 实例
（2） 设置 Blockchain 实例的 tip 为数据库中存储的最后一个块的哈希
4.	如果没有区块链：
（1） 创建创世块
（2） 存储到数据库
（3） 将创世块哈希保存为最后一个块的哈希
（4） 创建一个新的 Blockchain 实例，其 tip 指向创世块（tip 有尾部，尖端的意思，在这里 tip 存储的是最后一个块的哈希
*/
func NewBlockchain(address string) *Blockchain {
	//return &Blockchain{[]*block.Block{GenesisBlock()}}
	var tip []byte
	//打开一个数据库文件，如果文件不存在则创建该名字的文件
	db,err := bolt.Open(dbFile,0600,nil)
	if err != nil {
		log.Panic(err)
	}
	//读写操作数据库
	err = db.Update(func(tx *bolt.Tx) error{
		b := tx.Bucket([]byte(blocksBucket))
		//查看名字为blocksBucket的Bucket是否存在
		if b == nil {
			//不存在则从头 创建
			//fmt.Println(address,"!!!!!!")
			fmt.Println("There is no blockchain.Let's create one!")
			cbtx := NewCoinbaseTX(address, genesisCoinbaseData)
        	genesis := NewGenesisBlock(cbtx)//创建创世区块
			b, err := tx.CreateBucket([]byte(blocksBucket)) //创建名为blocksBucket的桶
			if err != nil {
				log.Panic(err)
			}
			err = b.Put(genesis.Hash, genesis.Serialize()) //写入键值对，区块哈希对应序列化后的区块
			if err != nil {
				log.Panic(err)
			}
			err = b.Put([]byte("l"),genesis.Hash) //"l"键对应区块链顶端区块的哈希
			if err != nil {
				log.Panic(err)
			}
			tip = genesis.Hash //指向最后一个区块，这里也就是创世区块
		} else {
			//如果存在blocksBucket桶，也就是存在区块链
			//通过键"l"映射出顶端区块的Hash值
			tip = b.Get([]byte("l"))
		}
 
		return nil
	})
 
	bc := Blockchain{tip,db}  //此时Blockchain结构体字段已经变成这样了
	return &bc
 
}
 
//为了防止区块链数据太大，我们一个一个地用区块链迭代器读取
type BlockchainIterator struct {
    currentHash []byte
    db          *bolt.DB
}

/*
每当要对链中的块进行迭代时，我们就会创建一个迭代器，里面存储了当前迭代的块哈希和数据库的连接
通过 db，迭代器逻辑上被附属到一个区块链上（这里的区块链指的是存储了一个数据库连接的 Blockchain 实例）
并且通过 Blockchain 方法进行创建
一个 tip 也就是区块链的一种标识符
*/
func (bc *Blockchain) Iterator() *BlockchainIterator {
    bci := &BlockchainIterator{bc.tip, bc.db}

    return bci
}
 
func (i *BlockchainIterator) Next() *Block {
    var block *Block

    err := i.db.View(func(tx *bolt.Tx) error {
        b := tx.Bucket([]byte(blocksBucket))
        encodedBlock := b.Get(i.currentHash)
        block = DeserializeBlock(encodedBlock)

        return nil
    })
    if err != nil {
		log.Panic(err)
	}
	//把迭代器中的当前区块哈希设置为上一区块的哈希，实现迭代的作用
    i.currentHash = block.PrevBlockHash

    return block
}

//通过交易ID找到一个交易
func (bc *Blockchain) FindTransaction(ID []byte) (Transaction,error) {
	bci := bc.Iterator()
	for {
		block := bci.Next()

		for _,tx := range block.Transactions {
			if bytes.Compare(tx.ID,ID) == 0 {
				return *tx,nil
			}
		}
		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
	return Transaction{},errors.New("Transaction is not found")
}

//对交易输入进行签名
func (bc *Blockchain) SignTransaction(tx *Transaction,privKey ecdsa.PrivateKey) {
	prevTXs := make(map[string]Transaction)
	for _,vin :=range tx.Vin {
		//fmt.Println(vin.Txid,"!!!!!!!")
		prevTX,err := bc.FindTransaction(vin.Txid) //找到输入引用的输出所在的交易
		if err != nil {
			log.Panic(err)
		}
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}
	tx.Sign(privKey,prevTXs)
}

//验证交易
func (bc *Blockchain) VerifyTransaction(tx *Transaction) bool {
	prevTXs := make(map[string]Transaction)
 
	for _, vin := range tx.Vin {
		prevTX,err := bc.FindTransaction(vin.Txid)
		if err != nil {
			log.Panic(err)
		}
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}
	return tx.Verify(prevTXs) //验证签名
}

//找到包含未花费输出的交易
//未花费交易输出（unspent transactions outputs, UTXO）
func (bc *Blockchain) FindUnspentTransactions(pubKeyHash []byte) []Transaction {
	var unspentTXs []Transaction
	spentTXOs := make(map[string][]int)
	bci := bc.Iterator()
  
	for {
		block := bci.Next()
  
		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)
  
		Outputs:
			for outIdx, out := range tx.Vout {
		  	// Was the output spent?
				//如果一个输出被一个地址锁定，并且这个地址恰好是我们要找的未花费交易输出的地址，那么这个输出就是我们想要的
				//不过在获取它之前，我们需要检查该输出是否已经被包含在一个输入中，也就是检查它是否已经被花费了  
				if spentTXOs[txID] != nil {
					for _, spentOut := range spentTXOs[txID] {
						if spentOut == outIdx {
						continue Outputs
			 			}
					}
		  		}
				//由于交易被存储在区块里，所以我们不得不检查区块链里的每一笔交易
				//从输出开始
				if out.IsLockedWithKey(pubKeyHash) {
					unspentTXs = append(unspentTXs, *tx)
		  		}
			}
		//检查完输出以后，我们将所有能够解锁给定地址锁定的输出的输入聚集起来
		//这并不适用于 coinbase 交易，因为它们不解锁输出
		if tx.IsCoinbase() == false {
		  for _, in := range tx.Vin {
			if in.UsesKey(pubKeyHash) {
			  inTxID := hex.EncodeToString(in.Txid)
			  spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Vout)
			}
		  }
		}
	  }
  
	  if len(block.PrevBlockHash) == 0 {
		break
	  }
	}
	//这个函数返回了一个交易列表，里面包含了未花费输出
	return unspentTXs
}

//为了计算余额，我们还需要一个函数将这些交易作为输入，然后仅返回一个输出切片
func (bc *Blockchain) FindUTXO(pubKeyHash []byte) []TXOutput {
	var UTXOs []TXOutput
	unspentTransactions := bc.FindUnspentTransactions(pubKeyHash)

	for _, tx := range unspentTransactions {
			for _, out := range tx.Vout {
					if out.IsLockedWithKey(pubKeyHash) {
							UTXOs = append(UTXOs, out)
					}
			}
	}

	return UTXOs
}

//现在，我们想要给其他人发送一些币。为此，我们需要创建一笔新的交易，将它放到一个块里，然后挖出这个块
//之前我们只实现了 coinbase 交易，现在我们需要一种通用的交易
func NewUTXOTransaction(from, to string, amount int, bc *Blockchain) *Transaction {
    var inputs []TXInput
    var outputs []TXOutput

    wallets,err := NewWallets()
	if err != nil {
		log.Panic(err)
	}
	_wallet := wallets.GetWallet(from)
	pubKeyHash := HashPubKey(_wallet.PublicKey)
	acc, validOutputs := bc.FindSpendableOutputs(pubKeyHash, amount)

	//fmt.Println(acc)

    if acc < amount {
        log.Panic("ERROR: Not enough funds")
    }

	//fmt.Println("!!!!")

    // Build a list of inputs
    for txid, outs := range validOutputs {
        txID, err := hex.DecodeString(txid)
		if err != nil {
			log.Panic(err)
		}
        for _, out := range outs {
            input := TXInput{txID,out,nil,_wallet.PublicKey}
            inputs = append(inputs, input)
        }
    }

    // Build a list of outputs
    outputs = append(outputs, *NewTXOutput(amount,to))
    if acc > amount {
        outputs = append(outputs, *NewTXOutput(acc - amount,from))
    }

    tx := Transaction{nil, inputs, outputs}
    tx.ID = tx.Hash()

	//fmt.Println(tx.ID)
	//fmt.Println(_wallet.PrivateKey)
	
	bc.SignTransaction(&tx, _wallet.PrivateKey)

    return &tx
}

//在创建新的输出前，我们首先必须找到所有的未花费输出，并且确保它们存储了足够的值
//我们创建两个输出：
//1.	一个由接收者地址锁定。这是给实际给其他地址转移的币。
//2.	一个由发送者地址锁定。这是一个找零。只有当未花费输出超过新交易所需时产生。记住：输出是不可再分的
//FindSpendableOutputs 方法基于之前定义的 FindUnspentTransactions 方法
func (bc *Blockchain) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
    unspentOutputs := make(map[string][]int)
    unspentTXs := bc.FindUnspentTransactions(pubKeyHash)
    accumulated := 0

Work:
    for _, tx := range unspentTXs {
        txID := hex.EncodeToString(tx.ID)

        for outIdx, out := range tx.Vout {
            if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
                accumulated += out.Value
                unspentOutputs[txID] = append(unspentOutputs[txID], outIdx)
				//对所有的未花费交易进行迭代，并对它的值进行累加
				//累加值大于或等于我们想要传送的值时，它就会停止并返回累加值，同时返回的还有通过交易 ID 进行分组的输出索引
				//我们并不想要取出超出需要花费的钱
                if accumulated >= amount {
                    break Work
                }
            }
        }
    }

    return accumulated, unspentOutputs
}