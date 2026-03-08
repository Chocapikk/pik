package sdk

// SleepCheck performs a sleep-based timing check. It calls fn with a random
// delay (2-4s) three times and verifies the response takes at least that long.
// Returns Vulnerable if 2+ rounds match, Safe otherwise.
func SleepCheck(run *Context, fn func(delay int) error) (CheckResult, error) {
	hits := 0
	for range 3 {
		delay := RandInt(2, 4)
		run.Elapsed(true)
		err := fn(delay)
		elapsed := run.Elapsed(false)
		if err != nil {
			continue
		}
		if elapsed >= float64(delay)-0.5 {
			hits++
		}
	}
	if hits >= 2 {
		return Vulnerable(Sprintf("command injection confirmed via sleep timing (%d/3)", hits))
	}
	return Safe("sleep timing check did not trigger")
}
