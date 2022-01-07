package blockchain

// TODO Placeholder for upcoming Block structure. // Needs a lot more information from Risto.
type Block struct {
	blockNr      uint64
	stateRoot    []byte
	transactions []Transaction
}

func (b *Block) appendTx(tx Transaction) {
	b.transactions = append(b.transactions, tx)
}
