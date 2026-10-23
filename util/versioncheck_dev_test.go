package util

import "testing"

func TestDevVersionIsNotOlderThanItsRelease(t *testing.T) {
	if VersionStringToNumeric("2.1.5-dev") < VersionStringToNumeric("2.1.5") {
		t.Fatal("2.1.5-dev must not be older than 2.1.5")
	}
	if VersionStringToNumeric("2.1.5-dev-gabc1234") != VersionStringToNumeric("2.1.5-dev") {
		t.Fatal("commit suffix must not change the version")
	}
	if VersionStringToNumeric("2.1.5-dev") >= VersionStringToNumeric("2.1.6") {
		t.Fatal("2.1.5-dev must be older than the next release")
	}
}
