# Verthash One-Click Miner

This program will create a Dogecoin wallet that only you have access to.  Encrypt your wallet with a password that you will not forget. It will then create the Verthash datafile and immediately begin mining Vertcoin. You will receive payouts in Dogecoin to your built-in Dogecoin wallet from the pool selected in Settings.  Additional payout currencies via Zergpool and zpool are available in Settings.  
    
### What are the fees?
   
  - This program and the [miner](https://github.com/CryptoGraphics/VerthashMiner) it runs are completely free and have no fee
  - [Zergpool.com](https://zergpool.com/) and [zpool.ca](https://www.zpool.ca/) charge a 0.5% fee while mining
  - [HashCryptos.com](https://www.hashcryptos.com/) charges no fee (DOGE only)
  - Standard minimum Dogecoin network fee for withdrawal from built-in wallet (usually < 0.001 DOGE)

### When do I get paid?

  - **Zergpool.com and zpool.ca**
    - Please check 'View Payout Information' for payout thresholds
  - **HashCryptos.com**
    - Every 24 hours with the ability to change payout frequency to every 12 hours (DOGE only)

### What coins does this support?

  - Dogecoin
  - Vertcoin
  - Binance Coin
  - Bitcoin
  - Bitcoin Cash
  - Cake
  - Dash
  - Digibyte
  - Ethereum
  - Ethereum Classic
  - Firo (Zcoin)
  - Flux (Zelcash)
  - Groestlcoin
  - Horizen (Zencash)
  - Komodo
  - LBRY Credits
  - Litecoin
  - Monero
  - Peercoin
  - Polygon(ERC20)
  - Raptoreum
  - Ravencoin
  - Shiba Inu(ERC20)
  - Tether(ERC20, TRC20, BEP20)
  - Tron
  - Verge 
  - Zcash
  - Additional coins may be added via PR following this [guide](https://github.com/vertiond/verthash-one-click-miner/blob/master/ADD-A-COIN.md)

The Verthash One-Click Miner mines [Vertcoin](https://vertcoin.org) and is functionally the same as [upstream](https://github.com/vertcoin-project/one-click-miner-vnext) utilizing a new data directory, `verthash-ocm`.  This is essentially a fancy wrapper for [VerthashMiner](https://github.com/CryptoGraphics/VerthashMiner) which is also open source.

This software is available for Windows and Linux.

## FAQ

### Which GPUs are supported?

Please refer to this list of [supported hardware.](https://github.com/CryptoGraphics/VerthashMiner#supported-hardware)

### I have an error message that reads 'Failure to configure' or 'Checks failed'

You may need to add an exclusion to your antivirus / Windows Defender.  The data directory is located at `%AppData%\verthash-ocm` on Windows or `~/.verthash-ocm` on Linux.

### My GPU is supported but an error messages reads 'no compatible GPUs'

Update your GPU drivers to the latest version.

### My GPU is not being utilized

The OCM is using all of your GPU resources.  To verify, check CUDA usage for Nvidia or Compute 0/1 usage for AMD.

### I selected HashCryptos.com but Expected Earnings says zero

Please make sure you have [activated your address.](https://www.hashcryptos.com/) It may take a few minutes to activate before you see Expected Earnings and hashrate.

## Building

The GUI is based on [Wails](https://wails.app) and [Go](https://golang.org/).

Install the Wails [prerequisites](https://wails.app/gettingstarted/) for your platform, and then run:

```bash
go get github.com/wailsapp/wails/cmd/wails
```

Then clone this repository, and inside its main folder, execute:

```bash
wails build
```

## Donations

If you want to support the further development of the upstream One Click Miner, feel free to donate Vertcoin to [Vmnbtn5nnNbs1otuYa2LGBtEyFuarFY1f8](https://insight.vertcoin.org/address/Vmnbtn5nnNbs1otuYa2LGBtEyFuarFY1f8).
