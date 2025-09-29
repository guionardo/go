package set

import "encoding/json"

// MarshalJSON produces a list serialization of the itens.
// The order of the itens is indetermined, so don't do comparisions between two JSONS
func (s Set[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.ToArray())
}

// UnmarshalJSON fills the set with data from JSON
func (s Set[T]) UnmarshalJSON(data []byte) error {
	tmpArr := make([]T, 0)
	err := json.Unmarshal(data, &tmpArr)
	if err == nil {
		s.Clear()
		s.AddMultiple(tmpArr...)
	}
	return err
}
