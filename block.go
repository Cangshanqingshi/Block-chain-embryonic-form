package main

import (
	"encoding/gob"
	"bytes"
	"log"
    "crypto/sha256"
)

//区块的结构体
type Block struct {
    Timestamp     int64//当前时间戳，即区块创建时间
    //Data          []byte//存储的信息
    Transactions  []*Transaction//交易，这里用的是数组，也就是说一个块能存多个交易
    PrevBlockHash []byte//前一个块的哈希
    Hash          []byte//当前块哈希
	Nonce         int//Nonce
}

//我们想要通过仅仅一个哈希，就可以识别一个块里面的所有交易。为此，我们获得每笔交易的哈希，将它们关联起来，然后获得一个连接后的组合哈希
func (b *Block) HashTransactions() []byte {
	var txHash [32]byte
	var txHashes [][]byte
	for _,tx := range b.Transactions {
		txHashes = append(txHashes,tx.Hash())
	}
	txHash = sha256.Sum256(bytes.Join(txHashes,[]byte{}))
 
	return txHash[:]
}
//比特币使用了一个更加复杂的技术：它将一个块里面包含的所有交易表示为一个 Merkle tree ，然后在工作量证明系统中使用树的根哈希（root hash）
//这个方法能够让我们快速检索一个块里面是否包含了某笔交易，即只需 root hash 而无需下载所有交易即可完成判断。

//实现 Block 的序列化方法（把块变成能存进数据库的字符串）
func (b *Block) Serialize() []byte {
    var result bytes.Buffer//定义一个 buffer 存储序列化之后的数据
    //初始化一个 gob encoder 并对 block 进行编码
    encoder := gob.NewEncoder(&result)
    err := encoder.Encode(b)
    if err != nil {
		log.Panic(err)
	}
    return result.Bytes()//结果作为一个字节数组返回
}
 
//解序列化的函数（把数据库里的字符串解出来）
func DeserializeBlock(d []byte) *Block {
    var block Block

    decoder := gob.NewDecoder(bytes.NewReader(d))
    err := decoder.Decode(&block)
    if err != nil {
		log.Panic(err)
	}
    return &block
}