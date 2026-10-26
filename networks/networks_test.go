package networks

import (
	"reflect"
	"testing"
	"time"
)

func TestSetBlockbookURLs(t *testing.T) {
	t.Cleanup(func() {
		SetBlockbookURLs(DefaultBlockbookURLs)
		SetNetwork()
	})
	SetNetwork()

	SetBlockbookURLs(nil)
	if !reflect.DeepEqual(Active.BlockbookURLs, DefaultBlockbookURLs) {
		t.Fatalf("an empty list replaced the defaults: %v", Active.BlockbookURLs)
	}

	remote := []string{"https://bb.example/"}
	SetBlockbookURLs(remote)
	if !reflect.DeepEqual(Active.BlockbookURLs, remote) {
		t.Fatalf("Active.BlockbookURLs = %v, want %v", Active.BlockbookURLs, remote)
	}

	SetNetwork()
	if !reflect.DeepEqual(Active.BlockbookURLs, remote) {
		t.Fatalf("SetNetwork lost the remote list: %v", Active.BlockbookURLs)
	}
}

func TestWaitForEndpointsWaitsForRelease(t *testing.T) {
	WaitForEndpoints() // never held: must not block

	release := HoldEndpoints()
	done := make(chan struct{})
	go func() {
		WaitForEndpoints()
		close(done)
	}()
	select {
	case <-done:
		t.Fatal("WaitForEndpoints returned before release")
	case <-time.After(50 * time.Millisecond):
	}
	release()
	release() // releasing twice is harmless
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("WaitForEndpoints still blocked after release")
	}
}
