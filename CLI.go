package main
 
import (
	"fmt"
	"os"
	"flag"
	"strconv"
	"log"
	//"github.com/boltdb/bolt"
)

//首先我们想要拥有这些命令 1.加入区块命令 2.打印区块链命令
 
//所有命令行相关的操作都会通过 CLI 结构进行处理
type CLI struct {
    //bc *Blockchain
}

//加入输入格式错误信息提示
func(cli *CLI) printUsage() {
	fmt.Println("Usage:")
	//fmt.Println("  addblock -data Blockdata")
	fmt.Println("  printchain //Print all the blocks of the blockchain")
	fmt.Println("  createwallet //creat a wallet with a pair of key inside")
	fmt.Println("  getbalance -address ADDRESS  //get the balance from address")
	fmt.Println("  listaddresses //Lists all addresses from the wallet file")
	fmt.Println("  createblockchain -address ADDRESS //creat a chain and the address can get coinbase")
	fmt.Println("  send -from FROM -to TO -amount AMOUNT //address from send amount coin to address to")
}
 
//判断命令行参数，如果没有输入参数则显示提示信息
func (cli *CLI) validateArgs() {
	if len(os.Args) < 2 {
		fmt.Println("Please input somesthing")
		os.Exit(1)
	}
}
 
// //加入区块函数调用
//func (cli *CLI) addBlock(data string) {
//    cli.bc.AddBlock(data)
//    fmt.Println("Success!")
//}
 
//打印区块链函数调用
func (cli *CLI) printChain() {
	/*var funny int = 0
	db,err := bolt.Open(dbFile,0600,nil)
	if err != nil {
		log.Panic(err)
	}
	err = db.Update(func(tx *bolt.Tx) error{
		b := tx.Bucket([]byte(blocksBucket))
		//查看名字为blocksBucket的Bucket是否存在
		if b == nil {
			fmt.Print("There is no blockchain! You must creat one and then print it.")
			funny = 1
		} 
		return nil
	})
	if funny == 0{*/
		//实例化一条链
		bc := NewBlockchain("")  //因为已经有了链，不会重新创建链，所以接收的address设置为空
		defer bc.Db().Close()

   		bci := bc.Iterator()

   		for {
    	    block := bci.Next()

			fmt.Printf("============= Block %x ============\n", block.Hash)
   	 	    fmt.Printf("Timestamp: %d\n", block.Timestamp)
			fmt.Printf("Prev. hash: %x\n", block.PrevBlockHash)
			//fmt.Printf("Hash: %x\n", block.Hash)
    	    //fmt.Printf("Data: %s\n", block.Data)
    	    pow := NewProofOfWork(block)
     	   fmt.Printf("PoW: %s\n", strconv.FormatBool(pow.Validate()))
     	   fmt.Println()

			for _,tx := range block.Transactions {
				transaction := (*tx).String()
				fmt.Printf("%s\n",transaction)
			}
			fmt.Printf("\n\n")
			if len(block.PrevBlockHash) == 0 {			
				break
        	}
    	}
}	
//}

//创建一条链
func (cli *CLI) createBlockchain(address string) {
	if !ValidateAddress(address) {
		fmt.Println(/*log.Panic(*/"ERROR: Address is not valid"/*)*/)
		return
	}
	
	bc := NewBlockchain(address)
	bc.Db().Close()
	fmt.Println("Done!")
}

//创建钱包函数
func (cli *CLI) createWallet() {
	wallets, _ := NewWallets()
	address := wallets.CreateWallet()
	wallets.SaveToFile()
	fmt.Printf("Your new address: %s\n", address)
}

//求账户余额（账户余额就是由账户地址锁定的所有未花费交易输出的总和）
func (cli *CLI) getBalance(address string) {
	if !ValidateAddress(address) {
		log.Panic("ERROR: Address is not valid")
	}

	bc := NewBlockchain(address)
	defer bc.Db().Close()
 
	balance := 0

	pubKeyHash := Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1:len(pubKeyHash)-4]
	//这里的4是校验位字节数，这里就不在其他包调过来了

	UTXOs := bc.FindUTXO(pubKeyHash)
 
	//遍历UTXOs中的交易输出out，得到输出字段out.Value,求出余额
	for _,out := range UTXOs {
		balance += out.Value
	}
 
	fmt.Printf("Balance of '%s':%d\n",address,balance)
}

//列出地址名单,钱包集合中的地址有哪些
func (cli *CLI) listAddresses() {
	wallets, err := NewWallets()
	if err != nil {
		log.Panic(err)
	}
	addresses := wallets.GetAddresses()
	for _, address := range addresses {
		fmt.Println(address)
	}
}

//之前，我们没有实现挖矿奖励，我们只有在创建区块链的时候coinbaseTX给了奖励，但是之后每一次挖矿都没有给出奖励
//所以我们要实现每一个区块被挖出后要给矿工一笔挖矿奖励的交易，挖矿奖励实际上就是一笔CoinbaseTX
//coinbase交易只有一个输出，我们实现挖矿奖励非常简单，把coinbase交易放在区块的Transactions的第一个位置就行了
//send方法
func (cli *CLI) send(from,to string,amount int) {
	if !ValidateAddress(from) {
		log.Panic("ERROR: Address is not valid")
	}
	if !ValidateAddress(to) {
		log.Panic("ERROR: Address is not valid")
	}
	
	//fmt.Println(from)

	bc := NewBlockchain(from)
	defer bc.Db().Close()
 
	tx := NewUTXOTransaction(from,to,amount,bc)
	//挖出一个包含该交易的区块,此时区块只有这一个交易
	bc.MineBlock([]*Transaction{tx})
	fmt.Println("Send success!")
}
//比特币并不是一连串立刻完成这些事情（不过我们的实现是这么做的）
//相反，它会将所有新的交易放到一个内存池中（mempool），然后当一个矿工准备挖出一个新块时，它就从内存池中取出所有的交易，创建一个候选块
//只有当包含这些交易的块被挖出来，并添加到区块链以后，里面的交易才开始确认。

//入口函数
func (cli *CLI) Run() {
	//判断命令行输入参数的个数，如果没有输入任何参数则打印提示输入参数信息
	cli.validateArgs()
	//实例化flag集合
	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	listAddressesCmd := flag.NewFlagSet("listaddresses", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)

	//注册flag标志符
	getBalanceAddress := getBalanceCmd.String("address", "", "The address to get balance for")
	createBlockchainAddress := createBlockchainCmd.String("address", "", "The address to send genesis block reward to")
	sendFrom := sendCmd.String("from", "", "Source wallet address")
	sendTo := sendCmd.String("to", "", "Destination wallet address")
	sendAmount := sendCmd.Int("amount", 0, "Amount to send")

	switch os.Args[1] {		//os.Args为一个保存输入命令的切片
	case "getbalance":
		err := getBalanceCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createblockchain":
		err := createBlockchainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createwallet":
		err := createWalletCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "listaddresses":
		err := listAddressesCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "printchain":
		err := printChainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	default:
		cli.printUsage()
		os.Exit(1)
	}
	
	//进入被解析出的命令，进一步操作
	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			os.Exit(1)
		}
		cli.getBalance(*getBalanceAddress)
	}
 
	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			createBlockchainCmd.Usage()
			os.Exit(1)
		}
		//fmt.Println(*createBlockchainAddress,"!!!!!")
		cli.createBlockchain(*createBlockchainAddress)
	}
 
	if createWalletCmd.Parsed() {
		cli.createWallet()
	}

	if listAddressesCmd.Parsed() {
		cli.listAddresses()
	}
	
	if printChainCmd.Parsed() {
		cli.printChain()
	}
 
	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount <= 0 {
			sendCmd.Usage()
			os.Exit(1)
		}
		cli.send(*sendFrom, *sendTo, *sendAmount)
	}
}