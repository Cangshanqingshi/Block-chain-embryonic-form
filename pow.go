package main

import (
	"fmt"
	"crypto/sha256"
	"bytes"
	"math/big"
	"math"
	"time"
    "log"
    "strconv"
    "encoding/binary"
)

//设置证明难度
const targetBits = 15//没有设置太高，主要是考虑调试时的时间成本

//定义结构体，包含区块结构体，目标难度target
type ProofOfWork struct {
    block  *Block
    target *big.Int//目标（求得的哈希值小于上界即有效）
}

//设置随机数变化最大范围
const maxNonce = math.MaxInt64

//编写NewProofOfWork()方法，新建ProofOfWork结构体
func NewProofOfWork(b *Block) *ProofOfWork {
    //NewInt创建一个值为x的*int
	target := big.NewInt(1)

	//Lsh为移位函数，将括号中的前一个数（1）左移后一个数位
    target.Lsh(target, uint(256-targetBits))

    pow := &ProofOfWork{b, target}

    return pow
}
 
//准备数据，将所有数据转化为切片，再进行拼接
func (pow *ProofOfWork) prepareData(nonce int) []byte {
    data := bytes.Join(
        [][]byte{
            pow.block.PrevBlockHash,
            pow.block.HashTransactions(),
            //这里被修改，把之前的Data字段修改成交易字段的哈希
            []byte(strconv.FormatInt(pow.block.Timestamp,10)),
			[]byte(strconv.FormatInt(targetBits,10)),
			[]byte(strconv.FormatInt(int64(nonce),10)),
        },    
		[]byte{},
    )

    return data
}

//将一个 int64 转化为一个切片，proofofwork中准备数据时调用
func IntToHex(num int64) []byte{
	buff := new(bytes.Buffer)
	err := binary.Write(buff,binary.BigEndian,num)
	if err != nil{
		log.Panic(err)
	}

	return buff.Bytes()
}

//工作量证明寻找有效哈希
func (pow *ProofOfWork) Run() (int, []byte) {
    var hashInt big.Int
    var hash [32]byte
    nonce := 0
    for nonce < maxNonce {
        data := pow.prepareData(nonce)
        hash = sha256.Sum256(data)
        hashInt.SetBytes(hash[:])
		
		//比较无符号整数大小，看Hash值是否小于目标值
        if hashInt.Cmp(pow.target) == -1 {
            fmt.Printf("\rPow success!\nhash:%x nonce:%v", hash, nonce)
            break
        } else {
            nonce++
        }
    }
    fmt.Print("\n")
    return nonce, hash[:]
}
 
//生成新块的函数，参数需要Data/交易与PrevBlockHash,返回一个指向区块结构体的指针
func NewBlock(transactions []*Transaction, prevBlockHash []byte) *Block {
    block := &Block{time.Now().Unix(), transactions, prevBlockHash, []byte{}, 0}
    //生成一个pow结构体
	pow := NewProofOfWork(block)
	//工作量证明——运行计算出符合条件的nonce,hash值
    nonce, hash := pow.Run()
	//将结果赋值给Block结构体
    block.Hash = hash[:]
    block.Nonce = nonce

    return block
}
 
//对结果进行验证，看是否满足工作量证明难度
func (pow *ProofOfWork) Validate() bool {
    var hashInt big.Int

    data := pow.prepareData(pow.block.Nonce)
    hash := sha256.Sum256(data)
    hashInt.SetBytes(hash[:])//变量由切片
    isValid := hashInt.Cmp(pow.target) == -1

    return isValid
}