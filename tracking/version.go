package tracking

// Version of unstamped builds: development on top of the latest release. Release
// builds get the real version stamped in through -ldflags. Bump this after every
// release so development builds don't offer an update to the release they came from.
var version = "2.1.6-dev"
