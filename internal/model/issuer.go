package model

import (
	"slices"
	"strings"

	"github.com/webitel/im-account-service/internal/errors"
)

var (
  // reserved (system) issuers
  knownIssuers = []string{
    "user", "contact",
    "bot", "script", "scheme",
    "app", "service", "server",
    // services
    "webitel",
    "engine",
    "workflow", "flow",
    // gateways
    "viber",
    "signal",
    "webitel",
    "telegram",
    "whatsapp",
    "facebook",
    "instagram",
  }
)

func isValidIssuer(issuer string) error {

  if len(issuer) == 0 {
    return errors.BadRequest(
			errors.Status("BAD_ISSUER"),
			errors.Message("identity: issuer( !string ); required"),
		)
  }

  // ASCII only 
  for i, c := range issuer {
    switch {
    // case IsDigit(c):
    // case IsLower(c):
    // case IsUpper(c):
    case ' ' < c && c <= '~':
      // ANY printable ASCII character ; no [white]space(s)
      continue
    }
    return errors.BadRequest(
			errors.Status("BAD_ISSUER"),
			errors.Message("identity: issuer( %s ); invalid character at position %d", issuer, (i + 1)),
		)
  }

  caseIgnoreMatch := func(known string) bool {
    return strings.EqualFold(known, issuer)
  }
  
  // reserved := true
  reserved := slices.ContainsFunc(
    knownIssuers, caseIgnoreMatch,
  )

	if reserved {
		return errors.BadRequest(
			errors.Status("BAD_ISSUER"),
			errors.Message("identity: issuer( %s ); reserved", issuer),
		)
	}

  // [ OK ]
  return nil
}