package pipe

// DoFunnel runs Do() for each provided slice of args.
// Returns a slice of all output slices, or the first error encountered.
func (p Pipe) DoFunnel(multiArgs [][]interface{}) ([][]interface{}, error) {
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
