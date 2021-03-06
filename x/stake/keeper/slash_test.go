package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/stake/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

// setup helper function
// creates two validators
func setupHelper(t *testing.T, amt int64) (sdk.Context, Keeper, types.Params) {
	// setup
	ctx, _, keeper := CreateTestInput(t, false, amt)
	params := keeper.GetParams(ctx)
	pool := keeper.GetPool(ctx)
	numVals := 3
	pool.LooseTokens = amt * int64(numVals)

	// add numVals validators
	for i := 0; i < numVals; i++ {
		validator := types.NewValidator(addrVals[i], PKs[i], types.Description{})
		validator, pool, _ = validator.AddTokensFromDel(pool, amt)
		keeper.SetPool(ctx, pool)
		keeper.UpdateValidator(ctx, validator)
		keeper.SetValidatorByPubKeyIndex(ctx, validator)
	}

	return ctx, keeper, params
}

// tests Revoke, Unrevoke
func TestRevocation(t *testing.T) {
	// setup
	ctx, keeper, _ := setupHelper(t, 10)
	addr := addrVals[0]
	pk := PKs[0]

	// initial state
	val, found := keeper.GetValidator(ctx, addr)
	require.True(t, found)
	require.False(t, val.GetRevoked())

	// test revoke
	keeper.Revoke(ctx, pk)
	val, found = keeper.GetValidator(ctx, addr)
	require.True(t, found)
	require.True(t, val.GetRevoked())

	// test unrevoke
	keeper.Unrevoke(ctx, pk)
	val, found = keeper.GetValidator(ctx, addr)
	require.True(t, found)
	require.False(t, val.GetRevoked())

}

// tests slashUnbondingDelegation
func TestSlashUnbondingDelegation(t *testing.T) {
	ctx, keeper, params := setupHelper(t, 10)
	fraction := sdk.NewRat(1, 2)

	// set an unbonding delegation
	ubd := types.UnbondingDelegation{
		DelegatorAddr:  addrDels[0],
		ValidatorAddr:  addrVals[0],
		CreationHeight: 0,
		// expiration timestamp (beyond which the unbonding delegation shouldn't be slashed)
		MinTime:        0,
		InitialBalance: sdk.NewCoin(params.BondDenom, 10),
		Balance:        sdk.NewCoin(params.BondDenom, 10),
	}
	keeper.SetUnbondingDelegation(ctx, ubd)

	// unbonding started prior to the infraction height, stake didn't contribute
	slashAmount := keeper.slashUnbondingDelegation(ctx, ubd, 1, fraction)
	require.Equal(t, int64(0), slashAmount.Int64())

	// after the expiration time, no longer eligible for slashing
	ctx = ctx.WithBlockHeader(abci.Header{Time: int64(10)})
	keeper.SetUnbondingDelegation(ctx, ubd)
	slashAmount = keeper.slashUnbondingDelegation(ctx, ubd, 0, fraction)
	require.Equal(t, int64(0), slashAmount.Int64())

	// test valid slash, before expiration timestamp and to which stake contributed
	oldPool := keeper.GetPool(ctx)
	ctx = ctx.WithBlockHeader(abci.Header{Time: int64(0)})
	keeper.SetUnbondingDelegation(ctx, ubd)
	slashAmount = keeper.slashUnbondingDelegation(ctx, ubd, 0, fraction)
	require.Equal(t, int64(5), slashAmount.Int64())
	ubd, found := keeper.GetUnbondingDelegation(ctx, addrDels[0], addrVals[0])
	require.True(t, found)
	// initialbalance unchanged
	require.Equal(t, sdk.NewCoin(params.BondDenom, 10), ubd.InitialBalance)
	// balance decreased
	require.Equal(t, sdk.NewCoin(params.BondDenom, 5), ubd.Balance)
	newPool := keeper.GetPool(ctx)
	require.Equal(t, int64(5), oldPool.LooseTokens-newPool.LooseTokens)
}

// tests slashRedelegation
func TestSlashRedelegation(t *testing.T) {
	ctx, keeper, params := setupHelper(t, 10)
	fraction := sdk.NewRat(1, 2)

	// set a redelegation
	rd := types.Redelegation{
		DelegatorAddr:    addrDels[0],
		ValidatorSrcAddr: addrVals[0],
		ValidatorDstAddr: addrVals[1],
		CreationHeight:   0,
		// expiration timestamp (beyond which the redelegation shouldn't be slashed)
		MinTime:        0,
		SharesSrc:      sdk.NewRat(10),
		SharesDst:      sdk.NewRat(10),
		InitialBalance: sdk.NewCoin(params.BondDenom, 10),
		Balance:        sdk.NewCoin(params.BondDenom, 10),
	}
	keeper.SetRedelegation(ctx, rd)

	// set the associated delegation
	del := types.Delegation{
		DelegatorAddr: addrDels[0],
		ValidatorAddr: addrVals[1],
		Shares:        sdk.NewRat(10),
	}
	keeper.SetDelegation(ctx, del)

	// started redelegating prior to the current height, stake didn't contribute to infraction
	validator, found := keeper.GetValidator(ctx, addrVals[1])
	require.True(t, found)
	slashAmount := keeper.slashRedelegation(ctx, validator, rd, 1, fraction)
	require.Equal(t, int64(0), slashAmount.Int64())

	// after the expiration time, no longer eligible for slashing
	ctx = ctx.WithBlockHeader(abci.Header{Time: int64(10)})
	keeper.SetRedelegation(ctx, rd)
	validator, found = keeper.GetValidator(ctx, addrVals[1])
	require.True(t, found)
	slashAmount = keeper.slashRedelegation(ctx, validator, rd, 0, fraction)
	require.Equal(t, int64(0), slashAmount.Int64())

	// test valid slash, before expiration timestamp and to which stake contributed
	oldPool := keeper.GetPool(ctx)
	ctx = ctx.WithBlockHeader(abci.Header{Time: int64(0)})
	keeper.SetRedelegation(ctx, rd)
	validator, found = keeper.GetValidator(ctx, addrVals[1])
	require.True(t, found)
	slashAmount = keeper.slashRedelegation(ctx, validator, rd, 0, fraction)
	require.Equal(t, int64(5), slashAmount.Int64())
	rd, found = keeper.GetRedelegation(ctx, addrDels[0], addrVals[0], addrVals[1])
	require.True(t, found)
	// initialbalance unchanged
	require.Equal(t, sdk.NewCoin(params.BondDenom, 10), rd.InitialBalance)
	// balance decreased
	require.Equal(t, sdk.NewCoin(params.BondDenom, 5), rd.Balance)
	// shares decreased
	del, found = keeper.GetDelegation(ctx, addrDels[0], addrVals[1])
	require.True(t, found)
	require.Equal(t, int64(5), del.Shares.RoundInt64())
	// pool bonded tokens decreased
	newPool := keeper.GetPool(ctx)
	require.Equal(t, int64(5), oldPool.BondedTokens-newPool.BondedTokens)
}

// tests Slash at a future height (must panic)
func TestSlashAtFutureHeight(t *testing.T) {
	ctx, keeper, _ := setupHelper(t, 10)
	pk := PKs[0]
	fraction := sdk.NewRat(1, 2)
	require.Panics(t, func() { keeper.Slash(ctx, pk, 1, 10, fraction) })
}

// tests Slash at the current height
func TestSlashAtCurrentHeight(t *testing.T) {
	ctx, keeper, _ := setupHelper(t, 10)
	pk := PKs[0]
	fraction := sdk.NewRat(1, 2)

	oldPool := keeper.GetPool(ctx)
	validator, found := keeper.GetValidatorByPubKey(ctx, pk)
	require.True(t, found)
	keeper.Slash(ctx, pk, ctx.BlockHeight(), 10, fraction)

	// read updated state
	validator, found = keeper.GetValidatorByPubKey(ctx, pk)
	require.True(t, found)
	newPool := keeper.GetPool(ctx)

	// power decreased
	require.Equal(t, sdk.NewRat(5), validator.GetPower())
	// pool bonded shares decreased
	require.Equal(t, sdk.NewRat(5).RoundInt64(), oldPool.BondedShares.Sub(newPool.BondedShares).RoundInt64())
}

// tests Slash at a previous height with an unbonding delegation
func TestSlashWithUnbondingDelegation(t *testing.T) {
	ctx, keeper, params := setupHelper(t, 10)
	pk := PKs[0]
	fraction := sdk.NewRat(1, 2)

	// set an unbonding delegation
	ubd := types.UnbondingDelegation{
		DelegatorAddr:  addrDels[0],
		ValidatorAddr:  addrVals[0],
		CreationHeight: 11,
		// expiration timestamp (beyond which the unbonding delegation shouldn't be slashed)
		MinTime:        0,
		InitialBalance: sdk.NewCoin(params.BondDenom, 4),
		Balance:        sdk.NewCoin(params.BondDenom, 4),
	}
	keeper.SetUnbondingDelegation(ctx, ubd)

	// slash validator for the first time
	ctx = ctx.WithBlockHeight(12)
	oldPool := keeper.GetPool(ctx)
	validator, found := keeper.GetValidatorByPubKey(ctx, pk)
	require.True(t, found)
	keeper.Slash(ctx, pk, 10, 10, fraction)

	// read updating unbonding delegation
	ubd, found = keeper.GetUnbondingDelegation(ctx, addrDels[0], addrVals[0])
	require.True(t, found)
	// balance decreased
	require.Equal(t, sdk.NewInt(2), ubd.Balance.Amount)
	// read updated pool
	newPool := keeper.GetPool(ctx)
	// bonded tokens burned
	require.Equal(t, int64(3), oldPool.BondedTokens-newPool.BondedTokens)
	// read updated validator
	validator, found = keeper.GetValidatorByPubKey(ctx, pk)
	require.True(t, found)
	// power decreased by 3 - 6 stake originally bonded at the time of infraction
	// was still bonded at the time of discovery and was slashed by half, 4 stake
	// bonded at the time of discovery hadn't been bonded at the time of infraction
	// and wasn't slashed
	require.Equal(t, sdk.NewRat(7), validator.GetPower())

	// slash validator again
	ctx = ctx.WithBlockHeight(13)
	keeper.Slash(ctx, pk, 9, 10, fraction)
	ubd, found = keeper.GetUnbondingDelegation(ctx, addrDels[0], addrVals[0])
	require.True(t, found)
	// balance decreased again
	require.Equal(t, sdk.NewInt(0), ubd.Balance.Amount)
	// read updated pool
	newPool = keeper.GetPool(ctx)
	// bonded tokens burned again
	require.Equal(t, int64(6), oldPool.BondedTokens-newPool.BondedTokens)
	// read updated validator
	validator, found = keeper.GetValidatorByPubKey(ctx, pk)
	require.True(t, found)
	// power decreased by 3 again
	require.Equal(t, sdk.NewRat(4), validator.GetPower())

	// slash validator again
	// all originally bonded stake has been slashed, so this will have no effect
	// on the unbonding delegation, but it will slash stake bonded since the infraction
	// this may not be the desirable behaviour, ref https://github.com/cosmos/cosmos-sdk/issues/1440
	ctx = ctx.WithBlockHeight(13)
	keeper.Slash(ctx, pk, 9, 10, fraction)
	ubd, found = keeper.GetUnbondingDelegation(ctx, addrDels[0], addrVals[0])
	require.True(t, found)
	// balance unchanged
	require.Equal(t, sdk.NewInt(0), ubd.Balance.Amount)
	// read updated pool
	newPool = keeper.GetPool(ctx)
	// bonded tokens burned again
	require.Equal(t, int64(9), oldPool.BondedTokens-newPool.BondedTokens)
	// read updated validator
	validator, found = keeper.GetValidatorByPubKey(ctx, pk)
	require.True(t, found)
	// power decreased by 3 again
	require.Equal(t, sdk.NewRat(1), validator.GetPower())

	// slash validator again
	// all originally bonded stake has been slashed, so this will have no effect
	// on the unbonding delegation, but it will slash stake bonded since the infraction
	// this may not be the desirable behaviour, ref https://github.com/cosmos/cosmos-sdk/issues/1440
	ctx = ctx.WithBlockHeight(13)
	keeper.Slash(ctx, pk, 9, 10, fraction)
	ubd, found = keeper.GetUnbondingDelegation(ctx, addrDels[0], addrVals[0])
	require.True(t, found)
	// balance unchanged
	require.Equal(t, sdk.NewInt(0), ubd.Balance.Amount)
	// read updated pool
	newPool = keeper.GetPool(ctx)
	// just 1 bonded token burned again since that's all the validator now has
	require.Equal(t, int64(10), oldPool.BondedTokens-newPool.BondedTokens)
	// read updated validator
	validator, found = keeper.GetValidatorByPubKey(ctx, pk)
	require.True(t, found)
	// power decreased by 1 again, validator is out of stake
	require.Equal(t, sdk.NewRat(0), validator.GetPower())
}

// tests Slash at a previous height with a redelegation
func TestSlashWithRedelegation(t *testing.T) {
	ctx, keeper, params := setupHelper(t, 10)
	pk := PKs[0]
	fraction := sdk.NewRat(1, 2)

	// set a redelegation
	rd := types.Redelegation{
		DelegatorAddr:    addrDels[0],
		ValidatorSrcAddr: addrVals[0],
		ValidatorDstAddr: addrVals[1],
		CreationHeight:   11,
		MinTime:          0,
		SharesSrc:        sdk.NewRat(6),
		SharesDst:        sdk.NewRat(6),
		InitialBalance:   sdk.NewCoin(params.BondDenom, 6),
		Balance:          sdk.NewCoin(params.BondDenom, 6),
	}
	keeper.SetRedelegation(ctx, rd)

	// set the associated delegation
	del := types.Delegation{
		DelegatorAddr: addrDels[0],
		ValidatorAddr: addrVals[1],
		Shares:        sdk.NewRat(6),
	}
	keeper.SetDelegation(ctx, del)

	// slash validator
	ctx = ctx.WithBlockHeight(12)
	oldPool := keeper.GetPool(ctx)
	validator, found := keeper.GetValidatorByPubKey(ctx, pk)
	require.True(t, found)
	keeper.Slash(ctx, pk, 10, 10, fraction)

	// read updating redelegation
	rd, found = keeper.GetRedelegation(ctx, addrDels[0], addrVals[0], addrVals[1])
	require.True(t, found)
	// balance decreased
	require.Equal(t, sdk.NewInt(3), rd.Balance.Amount)
	// read updated pool
	newPool := keeper.GetPool(ctx)
	// bonded tokens burned
	require.Equal(t, int64(5), oldPool.BondedTokens-newPool.BondedTokens)
	// read updated validator
	validator, found = keeper.GetValidatorByPubKey(ctx, pk)
	require.True(t, found)
	// power decreased by 2 - 4 stake originally bonded at the time of infraction
	// was still bonded at the time of discovery and was slashed by half, 4 stake
	// bonded at the time of discovery hadn't been bonded at the time of infraction
	// and wasn't slashed
	require.Equal(t, sdk.NewRat(8), validator.GetPower())

	// slash the validator again
	ctx = ctx.WithBlockHeight(12)
	validator, found = keeper.GetValidatorByPubKey(ctx, pk)
	require.True(t, found)
	keeper.Slash(ctx, pk, 10, 10, sdk.NewRat(3, 4))

	// read updating redelegation
	rd, found = keeper.GetRedelegation(ctx, addrDels[0], addrVals[0], addrVals[1])
	require.True(t, found)
	// balance decreased, now zero
	require.Equal(t, sdk.NewInt(0), rd.Balance.Amount)
	// read updated pool
	newPool = keeper.GetPool(ctx)
	// 7 bonded tokens burned
	require.Equal(t, int64(12), oldPool.BondedTokens-newPool.BondedTokens)
	// read updated validator
	validator, found = keeper.GetValidatorByPubKey(ctx, pk)
	require.True(t, found)
	// power decreased by 4
	require.Equal(t, sdk.NewRat(4), validator.GetPower())

	// slash the validator again, by 100%
	ctx = ctx.WithBlockHeight(12)
	validator, found = keeper.GetValidatorByPubKey(ctx, pk)
	require.True(t, found)
	keeper.Slash(ctx, pk, 10, 10, sdk.OneRat())

	// read updating redelegation
	rd, found = keeper.GetRedelegation(ctx, addrDels[0], addrVals[0], addrVals[1])
	require.True(t, found)
	// balance still zero
	require.Equal(t, sdk.NewInt(0), rd.Balance.Amount)
	// read updated pool
	newPool = keeper.GetPool(ctx)
	// four more bonded tokens burned
	require.Equal(t, int64(16), oldPool.BondedTokens-newPool.BondedTokens)
	// read updated validator
	validator, found = keeper.GetValidatorByPubKey(ctx, pk)
	require.True(t, found)
	// power decreased by 4, down to 0
	require.Equal(t, sdk.NewRat(0), validator.GetPower())

	// slash the validator again, by 100%
	// no stake remains to be slashed
	ctx = ctx.WithBlockHeight(12)
	validator, found = keeper.GetValidatorByPubKey(ctx, pk)
	require.True(t, found)
	keeper.Slash(ctx, pk, 10, 10, sdk.OneRat())

	// read updating redelegation
	rd, found = keeper.GetRedelegation(ctx, addrDels[0], addrVals[0], addrVals[1])
	require.True(t, found)
	// balance still zero
	require.Equal(t, sdk.NewInt(0), rd.Balance.Amount)
	// read updated pool
	newPool = keeper.GetPool(ctx)
	// no more bonded tokens burned
	require.Equal(t, int64(16), oldPool.BondedTokens-newPool.BondedTokens)
	// read updated validator
	validator, found = keeper.GetValidatorByPubKey(ctx, pk)
	require.True(t, found)
	// power still zero
	require.Equal(t, sdk.NewRat(0), validator.GetPower())
}

// tests Slash at a previous height with both an unbonding delegation and a redelegation
func TestSlashBoth(t *testing.T) {
	ctx, keeper, params := setupHelper(t, 10)
	fraction := sdk.NewRat(1, 2)

	// set a redelegation
	rdA := types.Redelegation{
		DelegatorAddr:    addrDels[0],
		ValidatorSrcAddr: addrVals[0],
		ValidatorDstAddr: addrVals[1],
		CreationHeight:   11,
		// expiration timestamp (beyond which the redelegation shouldn't be slashed)
		MinTime:        0,
		SharesSrc:      sdk.NewRat(6),
		SharesDst:      sdk.NewRat(6),
		InitialBalance: sdk.NewCoin(params.BondDenom, 6),
		Balance:        sdk.NewCoin(params.BondDenom, 6),
	}
	keeper.SetRedelegation(ctx, rdA)

	// set the associated delegation
	delA := types.Delegation{
		DelegatorAddr: addrDels[0],
		ValidatorAddr: addrVals[1],
		Shares:        sdk.NewRat(6),
	}
	keeper.SetDelegation(ctx, delA)

	// set an unbonding delegation
	ubdA := types.UnbondingDelegation{
		DelegatorAddr:  addrDels[0],
		ValidatorAddr:  addrVals[0],
		CreationHeight: 11,
		// expiration timestamp (beyond which the unbonding delegation shouldn't be slashed)
		MinTime:        0,
		InitialBalance: sdk.NewCoin(params.BondDenom, 4),
		Balance:        sdk.NewCoin(params.BondDenom, 4),
	}
	keeper.SetUnbondingDelegation(ctx, ubdA)

	// slash validator
	ctx = ctx.WithBlockHeight(12)
	oldPool := keeper.GetPool(ctx)
	validator, found := keeper.GetValidatorByPubKey(ctx, PKs[0])
	require.True(t, found)
	keeper.Slash(ctx, PKs[0], 10, 10, fraction)

	// read updating redelegation
	rdA, found = keeper.GetRedelegation(ctx, addrDels[0], addrVals[0], addrVals[1])
	require.True(t, found)
	// balance decreased
	require.Equal(t, sdk.NewInt(3), rdA.Balance.Amount)
	// read updated pool
	newPool := keeper.GetPool(ctx)
	// loose tokens burned
	require.Equal(t, int64(2), oldPool.LooseTokens-newPool.LooseTokens)
	// bonded tokens burned
	require.Equal(t, int64(3), oldPool.BondedTokens-newPool.BondedTokens)
	// read updated validator
	validator, found = keeper.GetValidatorByPubKey(ctx, PKs[0])
	require.True(t, found)
	// power not decreased, all stake was bonded since
	require.Equal(t, sdk.NewRat(10), validator.GetPower())
}
