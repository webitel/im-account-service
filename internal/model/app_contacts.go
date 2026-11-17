package model

import (
	"strings"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	"github.com/webitel/im-account-service/internal/errors"
	v1 "github.com/webitel/im-account-service/proto/gen/im/service/admin/v1"
)

// ValidateInputApp validates InputApp constraints that the proto declares via
// protovalidate but this service does not enforce automatically (no protovalidate
// interceptor is registered).
func ValidateInputApp(input *v1.InputApp) error {
	return ValidateContactAllowList(input.GetAllowContacts())
}

// ValidateContactAllowList enforces the rules declared on ContactAllowList and
// ContactAllowRule: contact_id must be a canonical 36-char UUID, contact_id values
// must be unique (case-insensitive), and at most one rule may be marked as default.
// A nil list (no restriction) and an empty list (nobody allowed) are both valid.
func ValidateContactAllowList(list *v1.ContactAllowList) error {
	var (
		seen         = make(map[string]int, len(list.GetRules()))
		defaultCount int
	)

	for i, rule := range list.GetRules() {
		id := rule.GetContactId()

		// protovalidate string.uuid accepts only the 8-4-4-4-12 hex form;
		// uuid.Parse also accepts urn:uuid:, {...} and 32-hex forms, hence the length check.
		if _, err := uuid.Parse(id); len(id) != 36 || err != nil {
			return errors.BadRequest(
				errors.Status("BAD_APP_ALLOW_CONTACTS"),
				errors.Message("app: allow_contacts.rules[%d].contact_id( %s ); invalid UUID", i, id),
			)
		}

		key := strings.ToLower(id)
		if first, dup := seen[key]; dup {
			return errors.BadRequest(
				errors.Status("BAD_APP_ALLOW_CONTACTS"),
				errors.Message("app: allow_contacts.rules[%d].contact_id( %s ); duplicate of rules[%d]", i, id, first),
			)
		}

		seen[key] = i

		if rule.GetDefault() {
			defaultCount++
			if defaultCount > 1 {
				return errors.BadRequest(
					errors.Status("BAD_APP_ALLOW_CONTACTS"),
					errors.Message("app: allow_contacts; at most one rule may be marked as default"),
				)
			}
		}
	}

	return nil
}

// normalizeContactAllowList returns a copy of list with every contact_id lower-cased,
// so that contact lookups by the gateway are a plain set match. The input is not mutated.
func normalizeContactAllowList(list *v1.ContactAllowList) *v1.ContactAllowList {
	if list == nil {
		return nil
	}

	out := proto.CloneOf(list)
	for _, rule := range out.GetRules() {
		rule.ContactId = strings.ToLower(rule.GetContactId())
	}

	return out
}
