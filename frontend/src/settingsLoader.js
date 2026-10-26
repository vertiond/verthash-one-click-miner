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
    b.GetAutoStart(),
    b.GetDebugging(),
    b.GetEnableIntegrated(),
    b.GetPools(),
    b.GetPool(),
    b.GetPayout(),
    b.GetNetwork(),
    b.GetCustomAddress(),
    b.Address(),
    b.GetPayoutAddresses(),
    b.GetLanguages(),
    b.GetLanguage(),
    b.GetSystemLocale()
  ])
    .then(function(r) {
      s.autoStart = r[0];
      s.debugging = r[1];
      s.enableIntegrated = r[2];
      s.pools = r[3];
      s.poolID = r[4];
      s.payoutID = r[5];
      s.network = r[6];
      s.customAddress = r[7];
      s.address = r[8];
      s.savedAddresses = r[9];
      s.languages = r[10];
      s.language = r[11];
      s.systemLocale = r[12];
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
