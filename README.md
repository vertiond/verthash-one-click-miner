# Verthash One-Click Miner

This program will create a Dogecoin wallet that only you have access to.  Encrypt your wallet with a password that you will not forget. It will then create the Verthash datafile and immediately begin mining Vertcoin. You will receive payouts in Dogecoin to your built-in Dogecoin wallet from the pool selected in Settings.  Additional payout currencies via zpool are available in Settings, where you can also pick the app's language.  
    
### What are the fees?
   
  - This program and the [miner](https://github.com/CryptoGraphics/VerthashMiner) it runs are completely free and have no fee
  - [zpool.ca](https://www.zpool.ca/) charges a 0.5% fee while mining
  - [HashCryptos.com](https://www.hashcryptos.com/) charges no fee (DOGE only)
  - Standard minimum Dogecoin network fee for withdrawal from built-in wallet (usually < 0.001 DOGE)

### When do I get paid?

  - **zpool.ca**
    - Please check 'View Payout Information' for payout thresholds
  - **HashCryptos.com**
    - Every 24 hours with the ability to change payout frequency to every 12 hours (DOGE only)

### What coins does this support?

  - Dogecoin
  - Vertcoin
  - Bitcoin
  - Bitcoin Cash
  - Dash
  - Digibyte
  - Firo
  - Flux
  - Groestlcoin
  - Horizen
  - Komodo
  - Litecoin
  - Monero
  - Peercoin
  - Ravencoin
  - Verge
  - Zcash

The pools and the payout coins each pool offers are loaded from [pools.json](pools.json) when the app starts, so they can change without a new release. Additional coins may be added via PR following this [guide](https://github.com/vertiond/verthash-one-click-miner/blob/master/ADD-A-COIN.md).

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

The GUI is based on [Wails](https://wails.app) v1 and [Go](https://golang.org/). This repository builds against a patched copy of Wails v1.16.9 in [third_party/wails](third_party/wails) (see [PATCHED.md](third_party/wails/PATCHED.md)), which `go.mod` uses in place of the published module.

You need:

  - Go 1.22 or later
  - Node.js (CI builds with Node 24). On Node 17 or later, set `NODE_OPTIONS=--openssl-legacy-provider` for the Vue CLI 4 frontend build
  - A C compiler for cgo: MinGW-w64 on Windows, or on Linux `libgtk-3-dev` and `libwebkit2gtk-4.0-dev` (Ubuntu 22.04 or earlier)
  - The Wails v1.16.9 CLI:

```bash
go install github.com/wailsapp/wails/cmd/wails@v1.16.9
```

Then clone this repository, and inside its main folder, execute:

```bash
wails build
```

Use `wails build -d` for a debug build, which writes what the app is doing to the console and to `debug.log` in the data directory.

## Donations

If you want to support the further development of the upstream One Click Miner, feel free to donate Vertcoin to [Vmnbtn5nnNbs1otuYa2LGBtEyFuarFY1f8](https://insight.vertcoin.org/address/Vmnbtn5nnNbs1otuYa2LGBtEyFuarFY1f8).
