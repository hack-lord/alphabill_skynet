package backend

import (
	"github.com/alphabill-org/alphabill/internal/txsystem"
	"github.com/alphabill-org/alphabill/internal/txsystem/money"
)

var (
	MoneyTxConverter = &TxConverter{}
	MoneySystemID    = []byte{0, 0, 0, 0}
)

type TxConverter struct {
}

func (t *TxConverter) ConvertTx(tx *txsystem.Transaction) (txsystem.GenericTransaction, error) {
	return money.NewMoneyTx(MoneySystemID, tx)
}
