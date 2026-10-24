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
	"github.com/vertiond/verthash-one-click-miner/networks"
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

func (m *Backend) setStringSetting(name string, value string) {
	setting := value
	m.settings.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(name, setting, nil)
		return err
	})
}

func (m *Backend) getStringSetting(name string) string {
	setting := ""
	m.settings.View(func(tx *buntdb.Tx) error {
		v, err := tx.Get(name)
		setting = v
		return err
	})
	return setting
}

func (m *Backend) GetPool() int {
	pool := m.getIntSetting("pool")
	if pool != 0 && !m.GetTestnet() && !pools.HasPool(pool, false) {
		// The saved pool was removed from the pool list, so use the default.
		logging.Warnf("Pool %d is no longer available, using the default pool\n", pool)
		pool = 0
	}
	if pool == 0 {
		if m.GetTestnet() {
			return 2 // Default P2Pool on testnet
		}
		// // Default to a random pool
		// rand.Seed(time.Now().UnixNano())
		// pools := pools.GetPools(m.GetTestnet())
		// pool := pools[rand.Intn(len(pools))].GetID()
		// // Save this setting immediately so that we don't get
		// // a different random pool in future calls to GetPool().
		// m.setIntSetting("pool", pool)
		// return pool
		if pools.HasPool(7, false) {
			return 7 // Default zpool
		}
		return pools.GetPools(false)[0].GetID()
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
	for _, p := range pools.GetPools(m.GetTestnet()) {
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
		if m.GetTestnet() {
			return payouts.VTCID // Default Vertcoin on testnet
		}
		return payouts.DOGEID // Default Dogecoin on mainnet
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
	selectedPool := pools.GetPool(selectedPoolID, m.GetTestnet())
	logging.Debugf("Selected pool: %s\n", selectedPool.GetName())
	for _, p := range selectedPool.GetPayouts(m.GetTestnet()) {
		pc = append(pc, PayoutChoice{
			ID:       p.GetID(),
			Name: p.GetDisplayName(),
		})
	}
	return pc
}

func (m *Backend) GetKickbackPayoutID(selectedPoolID int, selectedPayoutID int) int {
	selectedPool := pools.GetPool(selectedPoolID, m.GetTestnet())
	selectedPayout := pools.GetPayout(selectedPool, selectedPayoutID, m.GetTestnet())
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
	logging.Infof("Setting custom payout network to [%s]\n", network)
	m.network = network
	m.setStringSetting("network", network)
}

type NetworkChoice struct {
	Name string `json:"name"`
}

func (m *Backend) GetNetworks(selectedPoolID int, selectedPayoutID int) []NetworkChoice {
	nc := make([]NetworkChoice, 0)
	selectedPool := pools.GetPool(selectedPoolID, m.GetTestnet())
	selectedPayout := pools.GetPayout(selectedPool, selectedPayoutID, m.GetTestnet())
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

func (m *Backend) GetTestnet() bool {
	return false // Testnet is not necessary - return false
	//return m.getSetting("testnet")
}

func (m *Backend) SetTestnet(newTestnet bool) {
	if m.GetTestnet() != newTestnet {
		logging.Infof("Setting testnet to [%b]\n", newTestnet)
		m.setSetting("testnet", newTestnet)

		logging.Infof("Setting network to testnet=%b\n", newTestnet)
		networks.SetNetwork(newTestnet)

		logging.Infof("Calling WalletInitialized\n")
		m.WalletInitialized()
		logging.Infof("Done!")
	}
}

func (m *Backend) GetSkipVerthashExtendedVerify() bool {
	return false // Verification is fast and chain split is possible if datafile is wrong - return false
	//return m.getSetting("skipverthashverify")
}

func (m *Backend) SetSkipVerthashExtendedVerify(newVerthashVerify bool) {
	logging.Infof("Setting skip verthash verify to [%b]\n", newVerthashVerify)
	m.setSetting("skipverthashverify", newVerthashVerify)
}

func (m *Backend) GetClosedSource() bool {
	return false // No closed source Verthash miners - return false
	//return m.getSetting("closedsource")
}

func (m *Backend) SetClosedSource(newClosedSource bool) {
	logging.Infof("Setting closed source to [%b]\n", newClosedSource)
	m.setSetting("closedsource", newClosedSource)
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
	pool := pools.GetPool(poolID, m.GetTestnet())
	if r, ok := pool.(pools.DogeOnlyReporter); ok && r.ActivationURL() != "" {
		util.OpenBrowser(r.ActivationURL())
	}
}

// DogeOnlyPoolName returns the name of a pool other than the active one that
// pays out to the built-in Dogecoin wallet, or "" when there is none.
func (m *Backend) DogeOnlyPoolName() string {
	for _, p := range pools.GetPools(m.GetTestnet()) {
		if p.GetID() != m.pool.GetID() && pools.IsDogeOnly(p) {
			return p.GetName()
		}
	}
	return ""
}

func (m *Backend) GetVersion() string {
	return tracking.GetVersion()
}

func (m *Backend) PrerequisiteProxyLoop() {
	for pi := range m.prerequisiteInstall {
		send := "0"
		if pi {
			send = "1"
		}
		m.runtime.Events.Emit("prerequisiteInstall", send)
	}
}
