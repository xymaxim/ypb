package testutil

func PrintWantGot(diff string) string {
	return "(-want, +got):\n" + diff
}
