package tepra

// MMToPoints converts a length in mm to points.
func MMToPoints(mm float64) float64 {
	return mm * pointsPerMM
}

// PointsToMM converts a length in points to mm.
func PointsToMM(pt float64) float64 {
	return pt / pointsPerMM
}

// pointsPerMM converts mm to points.
const pointsPerMM = 72 / 25.4
