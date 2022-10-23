package main
 
import (
	"crypto/sha256"
	"encoding/gob"
	"bytes"
	"fmt"
	"log"
	"crypto/ecdsa"
	"encoding/hex"
	"crypto/rand"
	"math/big"
	"crypto/elliptic"
	"strings"
)

//比特币使用了一个叫做 Script 的脚本语言，用它来定义锁定和解锁输出的逻辑
//第一笔交易只有输出，没有输入
//当矿工挖出一个新的块时，它会向新的块中添加一个coinbase交易
//coinbase交易是一种特殊的交易，它不需要引用之前一笔交易的输出
//它“凭空”产生了币，这也是矿工获得挖出新块的奖励，可以理解为“发行新币”

const subsidy = 50  //挖矿奖励

//创建一个交易的数据结构，交易是由交易ID、交易输入、交易输出组成的,
//一个交易有多个输入和多个输出，所以这里的交易输入和输出应该是切片类型的
type Transaction struct {
    ID   []byte
    Vin  []TXInput	
		//输出里面存储了“币”
		//存储，指的是用一个数学难题对输出进行锁定
    Vout []TXOutput
}

//1.	有一些输出并没有被关联到某个输入上
//2.	一笔交易的输入可以引用之前多笔交易的输出
//3.	一个输入必须引用一个输出


//定义交易输出结构体
type TXOutput struct {
    Value        int	
		//币，value字段存储的是satoshi的数量
		//一个satoshi等于一百万分之一的BTC(0.00000001 BTC)
		//这也是比特币里面最小的货币单位
	PubkeyHash []byte
	//ScriptPubKey string	
		//对输出进行锁定
		//ScriptPubKey将会存储用户定义的钱包地址
}
//关于输出，非常重要的一点是：它们是不可再分的
//要么不用，如果要用，必须一次性用完
//如果它的值比需要的值大，那么就会产生一个找零，找零会返还给发送方

//定义交易输入结构体
type TXInput struct {
    Txid      []byte
		//一个输入引用了之前一笔交易的一个输出
		//Txid存储的是这笔之前的交易的ID,ID就是Transaction里面的字段
    Vout      int
		//存储的是该输出在这笔交易中所有输出的索引
		//因为一笔交易可能有多个输出，需要有信息指明是具体的哪一个
    Signature []byte
    PubKey    []byte
	//ScriptSig string
		//一个脚本，提供了可作用于一个输出的 ScriptPubKey 的数据
		//如果 ScriptSig 提供的数据是正确的，那么输出就会被解锁，然后被解锁的值就可以被用于产生新的输出
		//如果数据不正确，输出就无法被引用在输入中，或者说，也就是无法使用这个输出
		//这种机制，保证了用户无法花费属于其他人的币

		//由于我们还没有实现地址，所以ScriptSig将仅仅存储一个任意用户定义的钱包地址
}

//输出，就是 “币” 存储的地方
//每个输出都会带有一个解锁脚本，这个脚本定义了解锁该输出的逻辑
//每笔新的交易，必须至少有一个输入和输出
//一个输入引用了之前一笔交易的输出，并提供了数据（也就是 ScriptSig 字段）
//该数据会被用在输出的解锁脚本中解锁输出，解锁完成后即可使用它的值去产生新的输出

//创建一个coinbase交易
func NewCoinbaseTX(to, data string) *Transaction {
    if data == "" {
        data = fmt.Sprintf("Reward to '%s'", to)
    }
	//to代表此输出奖励给谁，一般都是矿工地址，data是交易附带的信息
	
	//fmt.Println(to)
	txin := TXInput{[]byte{},-1,nil,[]byte(data)}
    //txin := TXInput{[]byte{}, -1, data}
		//此交易中的交易输入,没有交易输入信息
		//Txid为空，Vout等于-1
	txout := NewTXOutput(subsidy,to)
    //txout := TXOutput{subsidy, to}
		//交易输出,subsidy为奖励矿工的币的数量
		//比特币中区块总数除以210000就是subsidy
		//挖出创世块的奖励是50BTC，每挖出210000个块后，奖励减半
		//在这里我们直接定义为一个常量
    tx := Transaction{nil, []TXInput{txin}, []TXOutput{*txout}}
    //tx.SetID()
	tx.ID = tx.Hash()
    return &tx
}

/*//设置交易ID，交易ID是序列化tx后再哈希
func (tx *Transaction) SetID() {
	var hash [32]byte
	var encoder bytes.Buffer

	enc := gob.NewEncoder(&encoder)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}
	hash = sha256.Sum256(encoder.Bytes())
	tx.ID =  hash[:]
}*/

//返回一个序列化后的交易
func (tx Transaction) Serialize() []byte {
	//var hash [32]byte
	var encoder bytes.Buffer
 
	enc := gob.NewEncoder(&encoder)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}
	//hash = sha256.Sum256(encoder.Bytes())
	//tx.ID =  hash[:]
	return encoder.Bytes()
}

//返回交易的哈希值
func (tx *Transaction) Hash() []byte {
	var hash [32]byte
 
	txCopy := *tx
	txCopy.ID = []byte{}
 
	hash = sha256.Sum256(txCopy.Serialize())
 
	return hash[:]
}

//1、每一个区块至少存储一笔coinbase交易，所以我们在区块的字段中把Data字段换成交易。
//2、把所有涉及之前Data字段都要换了，比如NewBlock()、GenesisBlock()、pow里的函数

/*
//在输入和输出上的锁定和解锁方法：
//在这里，我们只是将 script 字段与 unlockingData 进行了比较
//在后续文章我们基于私钥实现了地址以后，会对这部分进行改进
//输入上锁定的秘钥,表示能引用的输出是unlockingData
func (in *TXInput) CanUnlockOutputWith(unlockingData string) bool {
	return in.ScriptSig == unlockingData
}
//输出上的解锁秘钥,表示能被引用的输入是unlockingData
func (out *TXOutput) CanBeUnlockedWith(unlockingData string) bool {
	return out.ScriptPubKey == unlockingData 
}
*/

//判断是否为coinbase交易
func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vin[0].Vout == -1
}

//方法检查输入是否使用了指定密钥来解锁一个输出
func (in *TXInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := HashPubKey(in.PubKey)
 
	return bytes.Compare(lockingHash,pubKeyHash) == 0
}

//锁定交易输出到固定的地址，代表该输出只能由指定的地址引用
func (out *TXOutput) Lock(address []byte) {
	
	pubKeyHash := Base58Decode(address)
	//fmt.Println(address)
	pubKeyHash = pubKeyHash[1:len(pubKeyHash)-4]
	out.PubkeyHash = pubKeyHash 
}

//判断输入的公钥"哈希"能否解锁该交易输出
func (out *TXOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Compare(out.PubkeyHash,pubKeyHash) == 0
}

//创建一个新的交易输出
func NewTXOutput(value int,address string) *TXOutput {
	txo := &TXOutput{value,nil}
	txo.Lock([]byte(address))
 
	return txo
}

//对交易签名
//接受一个私钥和一个之前交易的 map
func (tx *Transaction) Sign(privKey ecdsa.PrivateKey,prevTXs map[string]Transaction) {
	//coinbase 交易因为没有实际输入，所以没有被签名
	if tx.IsCoinbase() {
		//fmt.Println("!!!!!!!")
		return
	}
	for _,vin := range tx.Vin {
		
		//fmt.Println(vin.Txid)
		
		if prevTXs[hex.EncodeToString(vin.Txid)].ID == nil {
			log.Panic("ERROR: Previous transaction is not correct")
		}
	}
	//将会被签署的是修剪后的交易副本，而不是一个完整交易
	//这个副本包含了所有的输入和输出，但是TXInput.Signature和TXIput.PubKey被设置为nil
	txCopy := tx.TrimmedCopy()
	//迭代副本中每一个输入
	//Signature被设置为nil(仅仅是一个双重检验)
	//PubKey被设置为所引用输出的PubKeyHash
	//除了当前交易，其他所有交易都是“空的”，也就是说他们的Signature和PubKey字段被设置为nil
	//输入是被分开签名的
	for inID,vin := range txCopy.Vin {
		prevTx := prevTXs[hex.EncodeToString(vin.Txid)]
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubkeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Vin[inID].PubKey = nil
		//通过privKey对txCopy.ID进行签名
		//一个 ECDSA 签名就是一对数字
		//将这对数字连接起来，并存储在输入的Signature字段
		r,s,err := ecdsa.Sign(rand.Reader,&privKey,txCopy.ID)
		if err != nil {
			log.Panic(err)
		}
		signature := append(r.Bytes(),s.Bytes()...)
 
		tx.Vin[inID].Signature = signature
	}
 
}

//创建在签名中修剪后的交易副本,之所以要这个副本是因为简化了输入交易本身的签名和公钥
func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TXInput
	var outputs []TXOutput
 
	for _,vin := range tx.Vin {
		inputs = append(inputs,TXInput{vin.Txid,vin.Vout,nil,nil})
	}
 
	for _,vout := range tx.Vout {
		outputs = append(outputs,TXOutput{vout.Value,vout.PubkeyHash})
	}
 
	txCopy := Transaction{tx.ID,inputs,outputs}
 
	return txCopy
}

//验证 交易输入的签名
func (tx *Transaction) Verify(prevTXs map[string]Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}
	for _,vin := range tx.Vin {
		//遍历输入交易，如果发现输入交易引用的上一交易的ID不存在，则Panic
		if prevTXs[hex.EncodeToString(vin.Txid)].ID == nil {
			log.Panic("ERROR: Previous transaction is not correct")
		}
	}

	txCopy := tx.TrimmedCopy() //修剪后的同一笔交易的副本
	curve := elliptic.P256() //椭圆曲线实例用于生成密钥对
 
	for inID,vin := range tx.Vin {
		prevTX := prevTXs[hex.EncodeToString(vin.Txid)]
		txCopy.Vin[inID].Signature = nil //双重验证
		txCopy.Vin[inID].PubKey = prevTX.Vout[vin.Vout].PubkeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Vin[inID].PubKey = nil
 
		r := big.Int{}
		s := big.Int{}
		sigLen := len(vin.Signature)
		r.SetBytes(vin.Signature[:(sigLen / 2)])
		s.SetBytes(vin.Signature[(sigLen / 2):])
 
		x := big.Int{}
		y := big.Int{}
		keyLen := len(vin.PubKey)
		x.SetBytes(vin.PubKey[:(keyLen / 2)])
		y.SetBytes(vin.PubKey[(keyLen / 2):])
 
		rawPubKey := ecdsa.PublicKey{curve,&x,&y}
		if ecdsa.Verify(&rawPubKey,txCopy.ID,&r,&s) == false {
			return false
		}
	}
	return true
}

//把交易转换成我们能正常读的形式
func (tx Transaction) String() string {
	var lines []string
	lines = append(lines, fmt.Sprintf("--Transaction %x:", tx.ID))
	for i, input := range tx.Vin {
		lines = append(lines, fmt.Sprintf(" -Input %d:", i))
		lines = append(lines, fmt.Sprintf("  TXID: %x", input.Txid))
		lines = append(lines, fmt.Sprintf("  Out:  %d", input.Vout))
		lines = append(lines, fmt.Sprintf("  Signature: %x", input.Signature))
		lines = append(lines, fmt.Sprintf("  PubKey:%x", input.PubKey))
	}
	for i, output := range tx.Vout {
		lines = append(lines, fmt.Sprintf(" -Output %d:", i))
		lines = append(lines, fmt.Sprintf("  Value: %d", output.Value))
		lines = append(lines, fmt.Sprintf("  Script: %x", output.PubkeyHash))
	}
	return strings.Join(lines,"\n")

}