package main
 
func main() {
	// bc := Blockchain.NewBlockchain()
	// defer bc.Db().Close()

	cli := CLI{}
	cli.Run()
}