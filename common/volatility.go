package common

func findMin(ii []float64, i, j int) (min float64) {
	min = ii[i]
	for i++; i < j; i++ {
		if ii[i] < min {
			min = ii[i]
		}
	}
	return
}

func findMax(ii []float64, i, j int) (max float64) {
	max = ii[i]
	for i++; i < j; i++ {
		if ii[i] > max {
			max = ii[i]
		}
	}
	return
}

func findMinMax(ii []float64, i, j int) (min float64, minIndex int, max float64, maxIndex int) {
	minIndex, maxIndex, min, max = i, i, ii[i], ii[i]
	for i++; i < j; i++ {
		if ii[i] < min {
			min, minIndex = ii[i], i
		} else if ii[i] >= max {
			max, maxIndex = ii[i], i
		}
	}
	return
}

func maxIntermediateGain(P []float64, i, j int, thr float64) float64 {
	min, minIndex, max, maxIndex := findMinMax(P, i, j)
	res := max/min - 1.0
	if minIndex <= maxIndex {
		return res
	}
	if res < thr {
		return 0.0
	}

	minL := findMin(P, i, maxIndex)
	rl := max/minL - 1.0
	if thr < rl {
		thr = rl
	}
	if minIndex+1 < j {
		maxR := findMax(P, minIndex+1, j)
		rr := maxR/min - 1.0
		if thr < rr {
			thr = rr
		}
	}
	rm := maxIntermediateGain(P, maxIndex+1, minIndex, thr)
	if thr < rm {
		thr = rm
	}
	return thr
}

func ComputeMIR(P []float64) float64 {
	min, minIndex, max, maxIndex := findMinMax(P, 0, len(P))
	thr := max/min - 1.0
	if minIndex <= maxIndex {
		return thr
	}
	loss := min/max - 1.0
	thr = -loss
	minL := findMin(P, 0, maxIndex)
	rl := max/minL - 1.0
	if thr < rl {
		thr = rl
	}
	if minIndex < len(P)-1 {
		maxR := findMax(P, minIndex+1, len(P))
		rr := maxR/min - 1.0
		if thr < rr {
			thr = rr
		}
	}
	rm := maxIntermediateGain(P, maxIndex+1, minIndex, thr)
	if thr < rm {
		thr = rm
	}
	if thr > -loss {
		return thr
	} else {
		return loss
	}
}
