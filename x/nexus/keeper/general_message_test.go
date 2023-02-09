package keeper_test

import (
	"fmt"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"

	"github.com/axelarnetwork/axelar-core/app"
	"github.com/axelarnetwork/axelar-core/testutils/rand"
	"github.com/axelarnetwork/axelar-core/utils"
	axelarnet "github.com/axelarnetwork/axelar-core/x/axelarnet/exported"
	evmtypes "github.com/axelarnetwork/axelar-core/x/evm/types"
	evmtestutils "github.com/axelarnetwork/axelar-core/x/evm/types/testutils"
	"github.com/axelarnetwork/axelar-core/x/nexus/exported"
	nexustestutils "github.com/axelarnetwork/axelar-core/x/nexus/exported/testutils"
	nexus "github.com/axelarnetwork/axelar-core/x/nexus/keeper"
	"github.com/axelarnetwork/utils/funcs"
	. "github.com/axelarnetwork/utils/test"
)

func TestSetNewGeneralMessage(t *testing.T) {
	var (
		generalMessage exported.GeneralMessage
		ctx            sdk.Context
		k              nexus.Keeper
	)
	cfg := app.MakeEncodingConfig()
	sourceChain := nexustestutils.RandomChain()
	sourceChain.Module = evmtypes.ModuleName
	destinationChain := nexustestutils.RandomChain()
	asset := rand.Coin()

	givenContractCallEvent := Given("a general message with token", func() {
		generalMessage = exported.GeneralMessage{
			ID: fmt.Sprintf("%s-%d", evmtestutils.RandomHash().Hex(), rand.PosI64()),

			Sender: exported.CrossChainAddress{
				Chain:   sourceChain,
				Address: evmtestutils.RandomAddress().Hex(),
			},
			Recipient: exported.CrossChainAddress{
				Chain:   destinationChain,
				Address: genCosmosAddr(destinationChain.Name.String()),
			},
			Status:      exported.Approved,
			PayloadHash: crypto.Keccak256Hash(rand.Bytes(int(rand.I64Between(1, 100)))).Bytes(),
			Asset:       &asset,
		}

		k, ctx = setup(cfg)
	})

	whenChainsAreRegistered := givenContractCallEvent.
		When("the source and destination chains are registered", func() {
			k.SetChain(ctx, sourceChain)
			k.SetChain(ctx, destinationChain)
		})

	errorWith := func(msg string) func(t *testing.T) {
		return func(t *testing.T) {
			assert.ErrorContains(t, k.SetNewMessage(ctx, generalMessage), msg)
		}
	}

	isCosmosChain := func(isCosmosChain bool) func() {
		return func() {
			if isCosmosChain {
				destChain := funcs.MustOk(k.GetChain(ctx, destinationChain.Name))
				destChain.Module = axelarnet.ModuleName
				k.SetChain(ctx, destChain)
			}
		}
	}

	isAssetRegistered := func(isRegistered bool) func() {
		return func() {
			if isRegistered {
				funcs.MustNoErr(k.RegisterAsset(ctx, sourceChain, exported.Asset{Denom: asset.Denom, IsNativeAsset: false}, utils.MaxUint, time.Hour))
				funcs.MustNoErr(k.RegisterAsset(ctx, destinationChain, exported.Asset{Denom: asset.Denom, IsNativeAsset: false}, utils.MaxUint, time.Hour))
			}
		}
	}

	givenContractCallEvent.
		When("the source chain is not registered", func() {}).
		Then("should return error", errorWith(fmt.Sprintf("source chain %s is not a registered chain", sourceChain.Name))).
		Run(t)

	givenContractCallEvent.
		When("the destination chain is not registered", func() {
			k.SetChain(ctx, sourceChain)
		}).
		Then("should return error", errorWith(fmt.Sprintf("destination chain %s is not a registered chain", destinationChain.Name))).
		Run(t)

	whenChainsAreRegistered.
		When("address validator for destination chain is set", isCosmosChain(true)).
		When("destination address is invalid", func() {
			generalMessage.Recipient.Address = rand.Str(20)
		}).
		Then("should return error", errorWith("decoding bech32 failed")).
		Run(t)

	whenChainsAreRegistered.
		When("address validator for destination chain is set", isCosmosChain(true)).
		When("asset is not registered", isAssetRegistered(false)).
		Then("should return error", errorWith("does not support foreign asset")).
		Run(t)

	whenChainsAreRegistered.
		When("address validator for destination chain is set", isCosmosChain(true)).
		When("asset is registered", isAssetRegistered(true)).
		Then("should succeed", func(t *testing.T) {
			assert.NoError(t, k.SetNewMessage(ctx, generalMessage))
		}).
		Run(t)
}