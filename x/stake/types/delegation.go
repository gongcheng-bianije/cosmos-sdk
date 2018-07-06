package types

import (
	"bytes"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Delegation represents the bond with tokens held by an account.  It is
// owned by one delegator, and is associated with the voting power of one
// pubKey.
type Delegation struct {
	DelegatorAddr sdk.AccAddress `json:"delegator_addr"`
	ValidatorAddr sdk.AccAddress `json:"validator_addr"`
	Shares        sdk.Rat        `json:"shares"`
	Height        int64          `json:"height"` // Last height bond updated
}

// two are equal
func (d Delegation) Equal(d2 Delegation) bool {
	return bytes.Equal(d.DelegatorAddr, d2.DelegatorAddr) &&
		bytes.Equal(d.ValidatorAddr, d2.ValidatorAddr) &&
		d.Height == d2.Height &&
		d.Shares.Equal(d2.Shares)
}

// ensure fulfills the sdk validator types
var _ sdk.Delegation = Delegation{}

// nolint - for sdk.Delegation
func (d Delegation) GetDelegator() sdk.AccAddress { return d.DelegatorAddr }
func (d Delegation) GetValidator() sdk.AccAddress { return d.ValidatorAddr }
func (d Delegation) GetBondShares() sdk.Rat       { return d.Shares }

//Human Friendly pretty printer
func (d Delegation) HumanReadableString() (string, error) {
	resp := "Delegation \n"
	resp += fmt.Sprintf("Delegator: %s\n", d.DelegatorAddr)
	resp += fmt.Sprintf("Validator: %s\n", d.ValidatorAddr)
	resp += fmt.Sprintf("Shares: %s", d.Shares.String())
	resp += fmt.Sprintf("Height: %d", d.Height)

	return resp, nil

}

//__________________________________________________________________

// element stored to represent the passive unbonding queue
type UnbondingDelegation struct {
	DelegatorAddr  sdk.AccAddress `json:"delegator_addr"`  // delegator
	ValidatorAddr  sdk.AccAddress `json:"validator_addr"`  // validator unbonding from owner addr
	CreationHeight int64          `json:"creation_height"` // height which the unbonding took place
	MinTime        int64          `json:"min_time"`        // unix time for unbonding completion
	InitialBalance sdk.Coin       `json:"initial_balance"` // atoms initially scheduled to receive at completion
	Balance        sdk.Coin       `json:"balance"`         // atoms to receive at completion
}

// nolint
func (d UnbondingDelegation) Equal(d2 UnbondingDelegation) bool {
	bz1 := MsgCdc.MustMarshalBinary(&d)
	bz2 := MsgCdc.MustMarshalBinary(&d2)
	return bytes.Equal(bz1, bz2)
}

//Human Friendly pretty printer
func (d UnbondingDelegation) HumanReadableString() (string, error) {
	resp := "Unbonding Delegation \n"
	resp += fmt.Sprintf("Delegator: %s\n", d.DelegatorAddr)
	resp += fmt.Sprintf("Validator: %s\n", d.ValidatorAddr)
	resp += fmt.Sprintf("Creation height: %v\n", d.CreationHeight)
	resp += fmt.Sprintf("Min time to unbond (unix): %v\n", d.MinTime)
	resp += fmt.Sprintf("Expected balance: %s", d.Balance.String())

	return resp, nil

}

//__________________________________________________________________

// element stored to represent the passive redelegation queue
type Redelegation struct {
	DelegatorAddr    sdk.AccAddress `json:"delegator_addr"`     // delegator
	ValidatorSrcAddr sdk.AccAddress `json:"validator_src_addr"` // validator redelegation source owner addr
	ValidatorDstAddr sdk.AccAddress `json:"validator_dst_addr"` // validator redelegation destination owner addr
	CreationHeight   int64          `json:"creation_height"`    // height which the redelegation took place
	MinTime          int64          `json:"min_time"`           // unix time for redelegation completion
	InitialBalance   sdk.Coin       `json:"initial_balance"`    // initial balance when redelegation started
	Balance          sdk.Coin       `json:"balance"`            // current balance
	SharesSrc        sdk.Rat        `json:"shares_src"`         // amount of source shares redelegating
	SharesDst        sdk.Rat        `json:"shares_dst"`         // amount of destination shares redelegating
}

// nolint
func (d Redelegation) Equal(d2 Redelegation) bool {
	bz1 := MsgCdc.MustMarshalBinary(&d)
	bz2 := MsgCdc.MustMarshalBinary(&d2)
	return bytes.Equal(bz1, bz2)
}

//Human Friendly pretty printer
func (d Redelegation) HumanReadableString() (string, error) {
	resp := "Redelegation \n"
	resp += fmt.Sprintf("Delegator: %s\n", d.DelegatorAddr)
	resp += fmt.Sprintf("Source Validator: %s\n", d.ValidatorSrcAddr)
	resp += fmt.Sprintf("Destination Validator: %s\n", d.ValidatorDstAddr)
	resp += fmt.Sprintf("Creation height: %v\n", d.CreationHeight)
	resp += fmt.Sprintf("Min time to unbond (unix): %v\n", d.MinTime)
	resp += fmt.Sprintf("Source shares: %s", d.SharesSrc.String())
	resp += fmt.Sprintf("Destination shares: %s", d.SharesDst.String())

	return resp, nil

}
