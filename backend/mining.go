package backend

import (
	"fmt"
	"time"

	"github.com/vertiond/verthash-one-click-miner/logging"
	"github.com/vertiond/verthash-one-click-miner/miners"
	"github.com/vertiond/verthash-one-click-miner/payouts"
	"github.com/vertiond/verthash-one-click-miner/pools"
	"github.com/vertiond/verthash-one-click-miner/tracking"
	"github.com/vertiond/verthash-one-click-miner/util"
)

func (m *Backend) GetArgs() miners.BinaryArguments {
	tracking.Track(tracking.TrackingRequest{
		Category: "Mining",
		Action:   "Switch Pool",
		Name:     fmt.Sprintf("%v", m.pool.GetName()),
	})

	return miners.BinaryArguments{
		StratumUrl:       m.pool.GetStratumUrl(),
		StratumUsername:  m.GetMiningAddress(),
		StratumPassword:  m.GetMiningPassword(),
		EnableIntegrated: m.getSetting("enableIntegrated"),
	}
}

func (m *Backend) GetPoolFee() string {
	return fmt.Sprintf("%0.2f%%", m.pool.GetFee())
}

func (m *Backend) GetPoolName() string {
	return m.pool.GetName()
}

func (m *Backend) GetPayoutTicker() string {
	if m.UseCustomPayout() {
		return m.payout.GetTicker()
	}
	return "DOGE"
}

// PayoutShortage is true when the pool reports it is short of the payout
// currency, so payouts in it may be delayed.
func (m *Backend) PayoutShortage() bool {
	reporter, ok := m.pool.(pools.ShortageReporter)
	if !ok {
		return false
	}
	return reporter.PayoutShortage(m.GetPayoutTicker())
}

func (m *Backend) GetMiningAddress() string {
	if m.UseCustomPayout() {
		return m.customAddress
	}
	return m.walletAddress
}

func (m *Backend) GetMiningPassword() string {
	network := m.network
	if m.ChainNetworkApplicableToPayout() {
		if network == "ERC20" {
			network = ""
		} else {
			network = fmt.Sprintf("-%s", m.network)
		}
	} else {
		network = ""
	}
	// We still may not use the chain network,
	// if the pool.GetPassword does not support it.
	return m.pool.GetPassword(m.GetPayoutTicker(), network)
}

func (m *Backend) PayoutInformation() {
	m.pool.OpenBrowserPayoutInfo(m.GetMiningAddress())
}

// formatHashrate renders a rate in hashes per second with the largest fitting
// unit, never going below kH/s.
func formatHashrate(hashesPerSecond float64) string {
	units := []string{"kH/s", "MH/s", "GH/s", "TH/s"}
	rate := hashesPerSecond / 1000
	unit := 0
	for rate >= 1000 && unit < len(units)-1 {
		rate /= 1000
		unit++
	}
	return fmt.Sprintf("%0.2f %s", rate, units[unit])
}

func (m *Backend) StartMining() bool {
	logging.Infof("Starting mining process...")

	tracking.Track(tracking.TrackingRequest{
		Category: "Mining",
		Action:   "Start",
	})

	args := m.GetArgs()

	startProcessMonitoring := make(chan bool)

	go func() {
		<-startProcessMonitoring
		continueLoop := true
		for continueLoop {
			newMinerBinaries := make([]*miners.BinaryRunner, 0)
			for _, br := range m.minerBinaries {
				if br.CheckRunning() == miners.RunningStateRapidFail {
					m.rapidFailures = append(m.rapidFailures, br)
					m.runtime.Events.Emit("minerRapidFail", br.MinerBinary.MainExecutableName)

				} else {
					newMinerBinaries = append(newMinerBinaries, br)
				}
			}

			m.minerBinaries = newMinerBinaries

			select {
			case <-m.stopMonitoring:
				continueLoop = false
			case <-time.After(time.Second):
			}
		}
		logging.Infof("Stopped monitoring thread")
	}()

	go func() {
		unitVtcPerBtc := 0.0
		unitPayoutCoinPerBtc := 0.0
		var myPayout payouts.Payout
		if m.UseCustomPayout() {
			myPayout = m.payout
		} else {
			// Default Dogecoin payout
			myPayout = payouts.DOGE()
		}

		cycles := 0
		var nhr uint64
		var th int64
		continueLoop := true
		for continueLoop {
			if cycles >= 600 {
				cycles = 0
			}
			if cycles == 0 {
				// Don't refresh this every time since we refresh it every second
				// and this pulls from Insight. Every 600s is fine (~every 4 blocks)
				nhr = util.GetNetHash()
				th = util.GetTipHeight()
				if myPayout.GetTicker() != "VTC" {
					unitVtcPerBtc = payouts.GetBitcoinPerUnitCoin("VTC")
					if myPayout.GetTicker() == "BTC" {
						unitPayoutCoinPerBtc = 1
					} else {
						time.Sleep(750 * time.Millisecond) // Put time between API calls
						unitPayoutCoinPerBtc = payouts.GetBitcoinPerUnitCoin(myPayout.GetTicker())
					}
					logging.Infof(fmt.Sprintf("Payout exchange rate: VTC/BTC=%0.16f, %s/BTC=%0.16f", unitVtcPerBtc, myPayout.GetTicker(), unitPayoutCoinPerBtc))
				}
			}
			cycles++

			hr := uint64(0)
			for _, br := range m.minerBinaries {
				hr += br.HashRate()
			}
			m.runtime.Events.Emit("hashRate", formatHashrate(float64(hr)))

			m.runtime.Events.Emit("networkHashRate", formatHashrate(float64(nhr)))

			coinsPerDay := util.GetCoinsPerDay(th)
			avgEarning := float64(hr) / float64(nhr) * float64(coinsPerDay)

			// Convert average earning from Vertcoin to selected payout coin
			avgEarningTicker := "VTC"
			if myPayout.GetTicker() != "VTC" {
				if unitVtcPerBtc != 0 && unitPayoutCoinPerBtc != 0 {
					avgEarningTicker = myPayout.GetTicker()
					avgEarning = avgEarning * unitVtcPerBtc / unitPayoutCoinPerBtc
				}
			}

			// Show at least three significant figures of average earning value
			avgEarningDecimals := 2
			if avgEarning > 0.0 && avgEarning < 1.0 {
				avgEarningScaled := avgEarning
				addDecimals := 1
				for addDecimals = 1; addDecimals <= 10; addDecimals++ {
					avgEarningScaled *= 10
					if avgEarningScaled >= 1.0 {
						break
					}
				}
				avgEarningDecimals += addDecimals
			}
			avgEarningStrfmt := "%0." + fmt.Sprint(avgEarningDecimals) + "f %s"

			m.runtime.Events.Emit("avgEarnings", fmt.Sprintf(avgEarningStrfmt, avgEarning, avgEarningTicker))

			select {
			case <-m.stopHash:
				continueLoop = false
			case <-m.refreshHashChan:
			case <-time.After(time.Second):
			}
		}
	}()

	go func() {
		continueLoop := true
		var pb uint64
		for continueLoop {
			m.wal.Update()
			b, bi := m.wal.GetBalance()
			m.runtime.Events.Emit("balance", fmt.Sprintf("%0.8f", float64(b)/float64(100000000)))
			m.runtime.Events.Emit("balanceImmature", fmt.Sprintf("%0.8f", float64(bi)/float64(100000000)))
			logging.Infof("Updating pending pool payout...")
			var payoutAddr string
			if m.UseCustomPayout() {
				payoutAddr = m.customAddress
			} else {
				payoutAddr = m.walletAddress
			}
			newPb := m.pool.GetPendingPayout(payoutAddr)
			pb = newPb
			m.runtime.Events.Emit("balancePendingPool", fmt.Sprintf("%0.8f", float64(pb)/float64(100000000)))
			select {
			case <-m.stopBalance:
				continueLoop = false
			case <-m.refreshBalanceChan:
			case <-time.After(time.Minute * 5):
			}
		}
	}()

	go func() {
		continueLoop := true
		for continueLoop {

			runningProcesses := 0
			for _, br := range m.minerBinaries {
				if br.IsRunning() {
					runningProcesses++
				}
			}

			m.runtime.Events.Emit("runningMiners", runningProcesses)

			timeout := time.Second * 1
			if runningProcesses > 0 {
				timeout = time.Second * 10
			}
			select {
			case <-m.stopRunningState:
				continueLoop = false
			case <-m.refreshRunningState:
			case <-time.After(timeout):
			}
		}
	}()

	for _, br := range m.minerBinaries {
		err := br.Start(args)
		if err != nil {
			m.StopMining()
			logging.Errorf("Failure to start %s: %s\n", br.MinerBinary.MainExecutableName, err.Error())
			return false
		}
	}

	startProcessMonitoring <- true

	return true
}

func (m *Backend) RefreshBalance() {

	m.refreshBalanceChan <- true
}

func (m *Backend) RefreshHashrate() {

	m.refreshHashChan <- true
}

func (m *Backend) RefreshRunningState() {

	m.refreshRunningState <- true
}

func (m *Backend) StopMining() bool {
	tracking.Track(tracking.TrackingRequest{
		Category: "Mining",
		Action:   "Stop",
	})
	select {
	case m.stopMonitoring <- true:
	default:
	}
	logging.Infof("Stopping mining process...")
	for _, br := range m.minerBinaries {
		err := br.Stop()
		if err != nil {
			logging.Errorf("Error stopping miner: %s %v", br.MinerBinary.MainExecutableName, err)
		}
	}
	select {
	case m.stopBalance <- true:
	default:
	}
	select {
	case m.stopHash <- true:
	default:
	}
	select {
	case m.stopRunningState <- true:
	default:
	}
	return true
}
