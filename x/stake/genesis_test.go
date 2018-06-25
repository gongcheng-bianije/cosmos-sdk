package stake

import (
	"testing"

	"github.com/stretchr/testify/assert"

	abci "github.com/tendermint/tendermint/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/stake/keeper"
)

func TestInitGenesis(t *testing.T) {
	vals := make([]Validator, 10)
	for i, _ := range vals {
		vals[i] = NewValidator(
			keeper.Addrs[i],
			keeper.PKs[i],
			Description{},
		)
		vals[i].PoolShares = PoolShares{
			Status: sdk.Bonded,
			Amount: sdk.NewRat(int64(i)),
		}
	}

	state := GenesisState{
		Validators: vals,
	}

	abcivals := make([]abci.Validator, len(vals))
	for i, val := range vals {
		abcivals[i] = sdk.ABCIValidator(val)
	}

	ctx, _, k := keeper.CreateTestInput(t, false, 100000)

	res := InitGenesis(ctx, k, state)

	assert.Equal(t, abcivals, res)
}
