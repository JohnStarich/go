package pipe

// Map runs p.Do() for each provided slice of args.
// Returns a slice of all output slices, or the first error encountered.
func Map(p Pipe, multiArgs [][]interface{}) ([][]interface{}, error) {
	results := make([][]interface{}, len(multiArgs))
	for i, args := range multiArgs {
		out, err := p.Do(args...)
		if err != nil {
			return nil, err
		}
		results[i] = out
	}
	return results, nil
}

// Filter runs p.Do() for each provided slice of args.
// Returns a slice of all successful output results.
// If none are successful, returns the last error.
func Filter(p Pipe, multiArgs [][]interface{}) ([][]interface{}, error) {
	var lastErr error
	results := make([][]interface{}, 0, len(multiArgs))
	for _, args := range multiArgs {
		out, err := p.Do(args...)
		if err != nil {
			lastErr = err
		} else {
			results = append(results, out)
		}
	}
	if len(results) == 0 {
		return nil, lastErr
	}
	return results, nil
}
