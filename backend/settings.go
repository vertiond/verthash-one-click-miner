package backend

import (
	"encoding/json"
	"fmt"
	// "math/rand"
	"strconv"
	"strings"
	// "time"

	"github.com/tidwall/buntdb"
	"github.com/vertiond/verthash-one-click-miner/logging"
	"github.com/vertiond/verthash-one-click-miner/payouts"
	"github.com/vertiond/verthash-one-click-miner/pools"
	"github.com/vertiond/verthash-one-click-miner/tracking"
	"github.com/vertiond/verthash-one-click-miner/util"
)

func (m *Backend) getSetting(name string) bool {
	setting := "0"
	err := m.settings.View(func(tx *buntdb.Tx) error {
		v, err := tx.Get(name)
		setting = v
		return err
	})
	if err != nil {
		logging.Errorf("Error in getSetting(%s): %v", name, err)
	}
	return setting == "1"
}

func (m *Backend) setSetting(name string, value bool) {
	setting := "0"
	if value {
		setting = "1"
	}
	err := m.settings.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(name, setting, nil)
		return err
	})
	if err != nil {
		logging.Errorf("Error in setSetting(%s): %v", name, err)
	}
}

func (m *Backend) setIntSetting(name string, value int) {
	setting := fmt.Sprintf("%d", value)
	err := m.settings.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(name, setting, nil)
		return err
	})
	if err != nil {
		logging.Errorf("Error in setSetting(%s): %v", name, err)
	}
}

func (m *Backend) getIntSetting(name string) int {
	setting := "0"
	err := m.settings.View(func(tx *buntdb.Tx) error {
		v, err := tx.Get(name)
		setting = v
		return err
	})
	if err != nil {
		logging.Errorf("Error in getSetting(%s): %v", name, err)
	}
	i, _ := strconv.Atoi(setting)
	return i
}

// getStringSetting returns "" for a setting that was never saved
func (m *Backend) getStringSetting(name string) string {
	setting := ""
	err := m.settings.View(func(tx *buntdb.Tx) error {
		v, err := tx.Get(name)
		setting = v
		return err
	})
	if err != nil && err != buntdb.ErrNotFound {
		logging.Errorf("Error in getSetting(%s): %v", name, err)
	}
	return setting
}

func (m *Backend) setStringSetting(name string, value string) {
	err := m.settings.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(name, value, nil)
		return err
	})
	if err != nil {
		logging.Errorf("Error in setSetting(%s): %v", name, err)
	}
}

func (m *Backend) GetPool() int {
	pool := m.getIntSetting("pool")
	if pool != 0 && !pools.HasPool(pool) {
		// The saved pool was removed from the pool list, so use the default.
		logging.Warnf("Pool %d is no longer available, using the default pool\n", pool)
		pool = 0
	}
	if pool == 0 {
		// // Default to a random pool
		// rand.Seed(time.Now().UnixNano())
		// pools := pools.GetPools()
		// pool := pools[rand.Intn(len(pools))].GetID()
		// // Save this setting immediately so that we don't get
		// // a different random pool in future calls to GetPool().
		// m.setIntSetting("pool", pool)
		// return pool
		if pools.HasPool(7) {
			return 7 // Default zpool
		}
		return pools.GetPools()[0].GetID()
	}
	return pool
}

func (m *Backend) SetPool(pool int) {
	if m.GetPool() != pool {
		m.setIntSetting("pool", pool)
		m.ResetPool()
		logging.Infof("Calling WalletInitialized\n")
		m.WalletInitialized()
		logging.Infof("Done!")
	}
}

func (m *Backend) SetEnableIntegrated(enabled bool) {
	m.setSetting("enableIntegrated", enabled)
}

func (m *Backend) GetEnableIntegrated() bool {
	return m.getSetting("enableIntegrated")
}

type PoolChoice struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	DogeOnly bool   `json:"dogeOnly"`
}

func (m *Backend) GetPools() []PoolChoice {
	pc := make([]PoolChoice, 0)
	for _, p := range pools.GetPools() {
		pc = append(pc, PoolChoice{
			ID:       p.GetID(),
			Name:     fmt.Sprintf("%s (%0.2f%% fee)", p.GetName(), p.GetFee()),
			DogeOnly: pools.IsDogeOnly(p),
		})
	}
	return pc
}

func (m *Backend) GetPayout() int {
	payout := m.getIntSetting("payout")
	if payout == 0 {
		return payouts.DOGEID // Default Dogecoin
	}
	return payout
}

func (m *Backend) SetPayout(payout int) {
	// Reset payout dropdown to Dogecoin if custom address is invalid
	if !m.ValidCustomAddress() {
		payout = payouts.DOGEID
	}
	if m.GetPayout() != payout {
		m.setIntSetting("payout", payout)
		m.ResetPayout()
	}
}

type PayoutChoice struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func (m *Backend) GetPayouts(selectedPoolID int) []PayoutChoice {
	pc := make([]PayoutChoice, 0)
	selectedPool := pools.GetPool(selectedPoolID)
	logging.Debugf("Selected pool: %s\n", selectedPool.GetName())
	for _, p := range selectedPool.GetPayouts() {
		pc = append(pc, PayoutChoice{
			ID:       p.GetID(),
			Name: p.GetDisplayName(),
		})
	}
	return pc
}

func (m *Backend) GetKickbackPayoutID(selectedPoolID int, selectedPayoutID int) int {
	selectedPool := pools.GetPool(selectedPoolID)
	selectedPayout := pools.GetPayout(selectedPool, selectedPayoutID)
	return selectedPayout.GetID()
}

func (m *Backend) GetCustomAddress() string {
	return m.getStringSetting("zergpoolAddress")
}

func (m *Backend) SetCustomAddress(newCustomAddress string) {
	logging.Infof("Setting custom address to [%s]\n", newCustomAddress)
	m.customAddress = newCustomAddress
	m.setStringSetting("zergpoolAddress", newCustomAddress)
}


const payoutAddressPrefix = "payoutAddress:"

// GetPayoutAddresses returns the payout address remembered for each payout ID,
// so switching the payout currency in settings doesn't lose the others.
func (m *Backend) GetPayoutAddresses() map[string]string {
	addresses := map[string]string{}
	m.settings.View(func(tx *buntdb.Tx) error {
		return tx.AscendKeys(payoutAddressPrefix+"*", func(key, value string) bool {
			addresses[strings.TrimPrefix(key, payoutAddressPrefix)] = value
			return true
		})
	})
	return addresses
}

// SetPayoutAddresses takes a JSON object of payout ID to address. An empty
// address forgets the one remembered for that payout.
func (m *Backend) SetPayoutAddresses(addressesJson string) {
	addresses := map[string]string{}
	if err := json.Unmarshal([]byte(addressesJson), &addresses); err != nil {
		logging.Errorf("Error in SetPayoutAddresses: %v", err)
		return
	}
	err := m.settings.Update(func(tx *buntdb.Tx) error {
		for payoutID, address := range addresses {
			key := payoutAddressPrefix + payoutID
			if address == "" {
				tx.Delete(key)
				continue
			}
			if _, _, err := tx.Set(key, address, nil); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		logging.Errorf("Error in SetPayoutAddresses: %v", err)
	}
}

func (m *Backend) PoolIsDogeOnly() bool {
	return pools.IsDogeOnly(m.pool)
}

func (m *Backend) PayoutIsVertcoin() bool {
	return (m.payout.GetID() == payouts.VTCID)
}

func (m *Backend) PayoutIsDogecoin() bool {
	return (m.payout.GetID() == payouts.DOGEID)
}

// TODO: Improve address validation
func (m *Backend) ValidCustomAddress() bool {
	customAddress := m.customAddress
	if customAddress != "" {
		return true
	}
	return false
}

func (m *Backend) UseCustomPayout() bool {
	// Use the custom payout config settings only if
	// - pool that pays out more than Dogecoin is selected
	// - non-Dogecoin payout option is selected
	// - address for payout is valid
	if (!m.PoolIsDogeOnly()) && (!m.PayoutIsDogecoin()) && m.ValidCustomAddress() {
		return true
	}
	return false
}

func (m *Backend) GetNetwork() string {
	network := m.getStringSetting("network")
	if network == "" {
		return "ERC20"
	}
	return network
}

func (m *Backend) SetNetwork(network string) {
	// Only payouts that list chain networks (like USDT) have one to save
	if len(m.payout.GetNetworks()) == 0 {
		return
	}
	logging.Infof("Setting custom payout network to [%s]\n", network)
	m.network = network
	m.setStringSetting("network", network)
}

type NetworkChoice struct {
	Name string `json:"name"`
}

func (m *Backend) GetNetworks(selectedPoolID int, selectedPayoutID int) []NetworkChoice {
	nc := make([]NetworkChoice, 0)
	selectedPool := pools.GetPool(selectedPoolID)
	selectedPayout := pools.GetPayout(selectedPool, selectedPayoutID)
	logging.Debugf("Selected payout: %s\n", selectedPayout.GetDisplayName())
	for _, n := range selectedPayout.GetNetworks() {
		nc = append(nc, NetworkChoice{
			Name: n,
		})
	}
	return nc
}

func (m *Backend) ChainNetworkApplicableToPayout() bool {
	network := m.network
	if network == "" {
		return false
	}
	network_in_payout := false
	payout_networks := m.payout.GetNetworks()
	for _, possible_network := range payout_networks {
		if network == possible_network {
			network_in_payout = true
			break
		}
	}
	return network_in_payout
}

func (m *Backend) GetDebugging() bool {
	return m.getSetting("debugging")
}

func (m *Backend) SetDebugging(newDebugging bool) {
	logging.Infof("Setting debugging to [%b]\n", newDebugging)
	m.setSetting("debugging", newDebugging)
}

func (m *Backend) GetAutoStart() bool {
	return util.GetAutoStart()
}

func (m *Backend) SetAutoStart(newAutoStart bool) {
	util.SetAutoStart(newAutoStart)
}

// ActivationInformation opens the page where the wallet address is activated
// for the given pool, if it has one.
func (m *Backend) ActivationInformation(poolID int) {
	pool := pools.GetPool(poolID)
	if r, ok := pool.(pools.DogeOnlyReporter); ok && r.ActivationURL() != "" {
		util.OpenBrowser(r.ActivationURL())
	}
}

// DogeOnlyPoolName returns the name of a pool other than the active one that
// pays out to the built-in Dogecoin wallet, or "" when there is none.
func (m *Backend) DogeOnlyPoolName() string {
	// A second copy that only shows the "already running" message has no pool
	if m.pool == nil {
		return ""
	}
	for _, p := range pools.GetPools() {
		if p.GetID() != m.pool.GetID() && pools.IsDogeOnly(p) {
			return p.GetName()
		}
	}
	return ""
}

func (m *Backend) GetVersion() string {
	return tracking.GetVersion()
}
