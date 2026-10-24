package p2poolgo

import "testing"

func TestChooseNetworkPrefersReadyNode(t *testing.T) {
	n := &Status{Ready: true, Difficulty: 24.8, Height: 200}
	d, tip, from := ChooseNetwork(n, 25.0, 199)
	if d != 24.8 || tip != 200 || !from {
		t.Fatalf("got %v %v %v", d, tip, from)
	}
}

func TestChooseNetworkNotReadyFallsBack(t *testing.T) {
	n := &Status{Ready: false, Difficulty: 24.8, Height: 200}
	d, tip, from := ChooseNetwork(n, 25.0, 199)
	if d != 25.0 || tip != 199 || from {
		t.Fatalf("got %v %v %v", d, tip, from)
	}
	d, tip, from = ChooseNetwork(nil, 25.0, 199)
	if d != 25.0 || tip != 199 || from {
		t.Fatalf("nil node: got %v %v %v", d, tip, from)
	}
}

func TestChooseNetworkRejects256x(t *testing.T) {
	n := &Status{Ready: true, Difficulty: 6348.8, Height: 200}
	d, tip, from := ChooseNetwork(n, 24.8, 199)
	if d != 24.8 || tip != 199 || from {
		t.Fatalf("got %v %v %v", d, tip, from)
	}
	n = &Status{Ready: true, Difficulty: 24.8 / 256, Height: 200}
	if _, _, from := ChooseNetwork(n, 24.8, 199); from {
		t.Fatal("tiny node difficulty accepted")
	}
}

func TestChooseNetworkZeroDifficultyFallsBack(t *testing.T) {
	n := &Status{Ready: true, Difficulty: 0, Height: 200}
	if d, tip, from := ChooseNetwork(n, 25.0, 199); d != 25.0 || tip != 199 || from {
		t.Fatalf("got %v %v %v", d, tip, from)
	}
	n = &Status{Ready: true, Difficulty: 24.8, Height: 0}
	if _, _, from := ChooseNetwork(n, 25.0, 199); from {
		t.Fatal("zero height accepted")
	}
}

func TestChooseNetworkNoBackendAcceptsNode(t *testing.T) {
	n := &Status{Ready: true, Difficulty: 6348.8, Height: 200}
	d, tip, from := ChooseNetwork(n, 0, 0)
	if d != 6348.8 || tip != 200 || !from {
		t.Fatalf("got %v %v %v", d, tip, from)
	}
}
