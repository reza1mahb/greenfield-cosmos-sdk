package feegrant

import (
	"time"

	"github.com/cosmos/gogoproto/proto"

	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const allowedMessagesListSizeLimitation = 54

var (
	_ FeeAllowanceI                 = (*AllowedMsgAllowance)(nil)
	_ types.UnpackInterfacesMessage = (*AllowedMsgAllowance)(nil)
)

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (a *AllowedMsgAllowance) UnpackInterfaces(unpacker types.AnyUnpacker) error {
	var allowance FeeAllowanceI
	return unpacker.UnpackAny(a.Allowance, &allowance)
}

// NewAllowedMsgAllowance creates new filtered fee allowance.
func NewAllowedMsgAllowance(allowance FeeAllowanceI, allowedMsgs []string) (*AllowedMsgAllowance, error) {
	msg, ok := allowance.(proto.Message)
	if !ok {
		return nil, sdkerrors.Wrapf(sdkerrors.ErrPackAny, "cannot proto marshal %T", msg)
	}
	any, err := types.NewAnyWithValue(msg)
	if err != nil {
		return nil, err
	}

	return &AllowedMsgAllowance{
		Allowance:       any,
		AllowedMessages: allowedMsgs,
	}, nil
}

// GetAllowance returns allowed fee allowance.
func (a *AllowedMsgAllowance) GetAllowance() (FeeAllowanceI, error) {
	allowance, ok := a.Allowance.GetCachedValue().(FeeAllowanceI)
	if !ok {
		return nil, sdkerrors.Wrap(ErrNoAllowance, "failed to get allowance")
	}

	return allowance, nil
}

// SetAllowance sets allowed fee allowance.
func (a *AllowedMsgAllowance) SetAllowance(allowance FeeAllowanceI) error {
	var err error
	a.Allowance, err = types.NewAnyWithValue(allowance.(proto.Message))
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrPackAny, "cannot proto marshal %T", allowance)
	}

	return nil
}

// Accept method checks for the filtered messages has valid expiry
func (a *AllowedMsgAllowance) Accept(ctx sdk.Context, fee sdk.Coins, msgs []sdk.Msg) (bool, error) {
	if !a.allMsgTypesAllowed(ctx, msgs) {
		return false, sdkerrors.Wrap(ErrMessageNotAllowed, "message does not exist in allowed messages")
	}

	allowance, err := a.GetAllowance()
	if err != nil {
		return false, err
	}

	remove, err := allowance.Accept(ctx, fee, msgs)
	if err == nil && !remove {
		if err = a.SetAllowance(allowance); err != nil {
			return false, err
		}
	}
	return remove, err
}

func (a *AllowedMsgAllowance) allowedMsgsToMap(ctx sdk.Context) map[string]bool {
	msgsMap := make(map[string]bool, len(a.AllowedMessages))
	for _, msg := range a.AllowedMessages {
		msgsMap[msg] = true
	}

	return msgsMap
}

func (a *AllowedMsgAllowance) allMsgTypesAllowed(ctx sdk.Context, msgs []sdk.Msg) bool {
	msgsMap := a.allowedMsgsToMap(ctx)

	for _, msg := range msgs {
		if !msgsMap[sdk.MsgTypeURL(msg)] {
			return false
		}
	}

	return true
}

// ValidateBasic implements FeeAllowance and enforces basic sanity checks
func (a *AllowedMsgAllowance) ValidateBasic() error {
	if a.Allowance == nil {
		return sdkerrors.Wrap(ErrNoAllowance, "allowance should not be empty")
	}
	if len(a.AllowedMessages) == 0 {
		return sdkerrors.Wrap(ErrNoMessages, "allowed messages shouldn't be empty")
	}
	if len(a.AllowedMessages) > allowedMessagesListSizeLimitation {
		return sdkerrors.Wrapf(ErrTooManyMessages, "allowed messages number: %d, limit: %d", len(a.AllowedMessages), allowedMessagesListSizeLimitation)
	}

	allowance, err := a.GetAllowance()
	if err != nil {
		return err
	}

	return allowance.ValidateBasic()
}

func (a *AllowedMsgAllowance) ExpiresAt() (*time.Time, error) {
	allowance, err := a.GetAllowance()
	if err != nil {
		return nil, err
	}
	return allowance.ExpiresAt()
}
