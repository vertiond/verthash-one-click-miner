package p2poolgo

// maxDifficultyRatio is the largest accepted factor between the node's
// difficulty and the last difficulty seen from the backend.
const maxDifficultyRatio = 16

// ChooseNetwork picks the network difficulty and tip height for the mining
// screen. node may be nil. lastBackendDiff == 0 means the backend /info was
// never reached. fromNode reports whether the node's numbers were used.
func ChooseNetwork(node *Status, lastBackendDiff float64, backendTip int64) (difficulty float64, tip int64, fromNode bool) {
	if node != nil && node.Ready && node.Difficulty > 0 && node.Height > 0 {
		if lastBackendDiff == 0 {
			return node.Difficulty, node.Height, true
		}
		r := node.Difficulty / lastBackendDiff
		if r >= 1.0/maxDifficultyRatio && r <= maxDifficultyRatio {
			return node.Difficulty, node.Height, true
		}
	}
	return lastBackendDiff, backendTip, false
}
