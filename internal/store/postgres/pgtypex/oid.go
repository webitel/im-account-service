package pgtypex

import "golang.org/x/exp/constraints"

// Reports whether given [oid] signed integer
// represents a valid positive value identifier
func IsValidOid[T constraints.Signed](oid T) bool {
  return oid > 0
}