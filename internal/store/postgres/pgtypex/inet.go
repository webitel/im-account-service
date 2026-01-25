package pgtypex

import (
	"errors"
	"fmt"
	"net"
	"net/netip"

	"github.com/jackc/pgx/v5/pgtype"
)

type NetIPValue net.IP

func (v NetIPValue) NetipPrefixValue() (netip.Prefix, error) {
	if v == nil {
		return netip.Prefix{}, nil
	}

	addr, ok := netip.AddrFromSlice([]byte(v))
	if !ok {
		return netip.Prefix{}, errors.New("invalid net.IP")
	}

	return netip.PrefixFrom(addr, addr.BitLen()), nil
}

type NetIPScanFunc func(v netip.Prefix) error

var _ pgtype.NetipPrefixScanner = NetIPScanFunc(nil)

func (scan NetIPScanFunc) ScanNetipPrefix(v netip.Prefix) error {
	if scan != nil {
		return scan(v)
	}
	// ignore
	return nil
}

func ScanNetIP(dst *net.IP) pgtype.NetipPrefixScanner {
	return NetIPScanFunc(func(src netip.Prefix) error {
		if !src.IsValid() {
			*dst = nil
			return nil
		}

		if src.Addr().BitLen() != src.Bits() {
			return fmt.Errorf("cannot scan %v into *net.IP", src)
		}

		*dst = net.IP(src.Addr().AsSlice())
		return nil
	})
}
