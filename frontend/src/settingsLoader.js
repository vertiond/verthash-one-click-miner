// Reads everything the settings screen shows from the backend. The calls that
// don't depend on each other run in parallel, and the last result is kept so
// the screen can show it right away the next time it is opened.
var last = null;

export function cachedSettings() {
  return last;
}

export function forgetSettings() {
  last = null;
}

export function loadSettings() {
  var b = window.backend.Backend;
  var s = {};
  return Promise.all([
    b.GetClosedSource(),
    b.GetAutoStart(),
    b.GetDebugging(),
    b.GetTestnet(),
    b.GetEnableIntegrated(),
    b.GetPools(),
    b.GetPool(),
    b.GetPayout(),
    b.GetNetwork(),
    b.GetCustomAddress(),
    b.Address(),
    b.GetPayoutAddresses()
  ])
    .then(function(r) {
      s.closedSourceMiner = r[0];
      s.autoStart = r[1];
      s.debugging = r[2];
      s.testnet = r[3];
      s.enableIntegrated = r[4];
      s.pools = r[5];
      s.poolID = r[6];
      s.payoutID = r[7];
      s.network = r[8];
      s.customAddress = r[9];
      s.address = r[10];
      s.savedAddresses = r[11];
      // These two depend on the selected pool and payout
      return Promise.all([
        b.GetPayouts(s.poolID),
        b.GetNetworks(s.poolID, s.payoutID)
      ]);
    })
    .then(function(r) {
      s.payouts = r[0];
      s.networks = r[1];
      last = s;
      return s;
    });
}
