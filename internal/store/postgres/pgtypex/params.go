package pgtypex

import (
	"fmt"
	"strconv"
)

// Parameters used to bind SQL query arguments
type Params map[string]any // pgx.NamedArgs

// Set protects against [re]write parameter with a new value
func (vs *Params) Set(param string, value any) error {
	data := (*vs)
	if data == nil {
		data = make(Params)
		(*vs) = data
	}
	if param == "" {
		param = strconv.Itoa(len(data) + 1)
	}
	if v, ok := data[param]; ok {
		if v != value {
			return fmt.Errorf("reset param[%q] value not allowed", param)
		}
		// OK ; skip due to the same value
		return nil
	}
	// SET
	data[param] = value
	return nil
}
