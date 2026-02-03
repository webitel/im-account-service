package handler

import (
	"cmp"
	"context"
	"encoding/json"
	"slices"
	"strconv"

	"github.com/webitel/im-account-service/internal/errors"
	"github.com/webitel/im-account-service/internal/model"

	impb "github.com/webitel/im-account-service/proto/gen/im/service/contact/v1"
)

type SearchContactRequest = impb.SearchContactRequest
type ContactSearchOption func(req *SearchContactRequest) error

func FindContactDc(dc int64) ContactSearchOption {
	return func(req *SearchContactRequest) error {
		req.DomainId = int32(dc)
		return nil
	}
}

func FindContactId(id string) ContactSearchOption {
	return func(req *SearchContactRequest) error {
		// if id == "" {
		// 	return nil // invalid ; set to return empty result
		// }
		if slices.Contains(req.Ids, id) {
			return nil // already present
		}
		req.Ids = append(req.Ids, id)
		return nil
	}
}

func FindContactIssuer(iss string) ContactSearchOption {
	return func(req *SearchContactRequest) error {
		// if iss == "" {
		// 	return nil // invalid
		// }
		if slices.Contains(req.IssId, iss) {
			return nil // already present
		}
		req.IssId = append(req.IssId, iss)
		return nil
	}
}

func FindContactSubject(iss, sub string) ContactSearchOption {
	return func(req *SearchContactRequest) error {
		// if sub == "" {
		// 	return nil // invalid
		// }

		err := FindContactIssuer(iss)(req)
		if err != nil {
			return err
		}

		if slices.Contains(req.Subjects, sub) {
			return nil // already present
		}
		req.Subjects = append(req.Subjects, sub)
		return nil
	}
}

func (srv *Service) GetContact(ctx context.Context, lookup ...ContactSearchOption) (*model.Contact, error) {

	// perform lookup by dc:iss/sub
	repo := srv.opts.Contacts
	req := &impb.SearchContactRequest{
		Page: 1,
		Size: 1,
	}

	for _, option := range lookup {
		err := option(req)
		if err != nil {
			return nil, err
		}
	}

	list, err := repo.SearchContact(ctx, req)

	if err != nil {
		return nil, err
	}

	var src *impb.Contact

	size := len(list.GetContacts())
	if list.GetNext() || size > 1 {
		return nil, errors.New(
			errors.Code(409),
			errors.Status("CONFLICT"),
			errors.Message("contact: too many records found"),
		)
	}

	if size == 1 {
		src = list.Contacts[0]
		// if src.GetIssId() != set.Iss || src.GetSubject() != set.Sub {
		// 	src = nil // NOT FOUND ; sanitize
		// }
	}

	var res model.Contact
	if src == nil || !contactFromProtoV1(src, &res) {
		return nil, nil // not found
	}

	return &res, nil
}

func (srv *Service) AddContact(ctx context.Context, set *model.Contact) error {
	// TODO: Client.Service("im-contact-service").SaveContact(set)
	repo := srv.opts.Contacts
	dst, err := repo.Upsert(
		ctx, &impb.CreateContactRequest{

			DomainId: int32(set.Dc),
			AppId:    set.App,

			IssId:   set.Iss,
			Subject: set.Sub,

			Type:     cmp.Or(set.Type, set.Iss),
			Name:     set.Name,
			Username: cmp.Or(set.Username, set.Name),

			Metadata: contactMdFormProtoV1(set),
		},
	)

	/*list, err := repo.SearchContact(
		ctx, &adv1.SearchContactRequest{
			Page: 1,
			Size: 1,
			// Q:        "",
			// Sort:     "",
			IssId:    []string{set.Iss},
			Subjects: []string{set.Sub},
			// Fields:   []string{},
			// AppId:    []string{},
			// Type:     []string{},
			// Ids:      []string{},
		},
	)

	if err != nil {
		return err
	}

	var src, dst *adv1.Contact
	size := len(list.GetContacts())

	if list.GetNext() || size > 1 {
		return errors.New(
			errors.Code(409),
			errors.Status("CONFLICT"),
			errors.Message("contact( %s@%s ); too many records found", set.Sub, set.Iss),
		)
	}

	if size == 1 {
		src = list.Contacts[0]
		if src.GetIssId() != set.Iss || src.GetSubject() != set.Sub {
			src = nil // NOT FOUND ; sanitize
		}
	}

	if src == nil {
		// CREATE
		dst, err = repo.CreateContact(
			ctx, &adv1.CreateContactRequest{

				DomainId: int32(set.Dc),
				AppId:    set.App,

				IssId:   set.Iss,
				Subject: set.Sub,

				Type:     cmp.Or(set.Type, set.Iss),
				Name:     set.Name,
				Username: cmp.Or(set.Username, set.Name),

				Metadata: contactMdFormProtoV1(set),
			},
		)
	} else {
		// UPDATE
		dst, err = repo.UpdateContact(
			ctx, &adv1.UpdateContactRequest{
				Id:       src.Id,
				Subject:  set.Sub, // CHANGE: DISALLOW !
				Name:     set.Name,
				Username: cmp.Or(set.Username, set.Name),
				Metadata: contactMdFormProtoV1(set),
			},
		)
	}*/

	if err != nil {
		return err
	}

	var res model.Contact
	if dst == nil || !contactFromProtoV1(dst, &res) {
		return errors.New(
			errors.Code(500),
			errors.Status("INTERNAL"),
			errors.Message("contact( %s@%s ); failed to refresh record", set.Sub, set.Iss),
		)
	}

	// refresh from persistent source
	(*set) = res
	return nil
}

func contactFormProtoV1(src *model.Contact) (dst *impb.Contact) {

	if src == nil {
		return nil
	}

	dst = &impb.Contact{

		Id:       src.Id,
		AppId:    src.App,
		DomainId: int32(src.Dc),

		IssId:   src.Iss,
		Subject: src.Sub,

		Type:     src.Type,
		Name:     src.Name,
		Username: src.Username,

		Metadata: contactMdFormProtoV1(src),

		CreatedAt: 0,
		UpdatedAt: 0,
	}

	return dst
}

func contactFromProtoV1(src *impb.Contact, dst *model.Contact) bool {

	if src == nil {
		return false
	}

	// sanitize
	(*dst) = model.Contact{

		Dc:  max(int64(src.DomainId), 0),
		Id:  src.Id,
		App: src.AppId,

		Iss: src.IssId,
		Sub: src.Subject,

		Type:     src.Type,
		Name:     src.Name,
		Username: src.Username,

		Metadata: nil, // below

		CreatedAt: model.Timestamp.Date(src.CreatedAt),
	}

	// if src.CreatedAt > 0 {
	// 	dst.CreatedAt = model.Timestamp.Date(src.CreatedAt)
	// }

	if src.UpdatedAt > 0 {
		date := model.Timestamp.Date(src.UpdatedAt)
		dst.UpdatedAt = &date
	}

	contactMdFromProtoV1(src, dst)

	return true
}

func contactMdFromProtoV1(src *impb.Contact, dst *model.Contact) {
	metadata := make(map[string]any, len(src.GetMetadata()))
	for k, vs := range src.GetMetadata() {
		switch k {
		// case ".dc":
		// 	dst.Dc, _ = strconv.ParseInt(vs, 10, 64)
		case ".given_name":
			dst.GivenName = vs
		case ".middle_name":
			dst.MiddleName = vs
		case ".family_name":
			dst.FamilyName = vs
		// case ".nickname":
		// 	dst.Nickname = vs
		// case ".preferred_username":
		// 	dst.PreferredUsername = vs
		case ".profile":
			dst.Profile = vs
		case ".picture":
			dst.Picture = vs
		case ".email":
			dst.Email = vs
		case ".email_verified":
			dst.EmailVerified, _ = strconv.ParseBool(vs)
		case ".phone_number":
			dst.PhoneNumber = vs
		case ".phone_number_verified":
			dst.PhoneNumberVerified, _ = strconv.ParseBool(vs)
		case ".gender":
			dst.Gender = vs
		case ".birthdate":
			dst.Birthdate = vs
		case ".zoneinfo":
			dst.Zoneinfo = vs
		case ".locale":
			dst.Locale = vs
		default:
			{
				var v any
				_ = json.Unmarshal([]byte(vs), &v)
				if v == nil {
					v = vs // keep as original string
				}
				metadata[k] = v
			}
		}
	}

	if len(metadata) == 0 {
		metadata = nil
	}
	dst.Metadata = metadata
}

func contactMdFormProtoV1(src *model.Contact) (metadata map[string]string) {

	boolValue := func(v bool) string {
		if v {
			return "true"
		}
		return ""
	}
	// uint64Value := func(v int64) string {
	// 	if v > 0 {
	// 		return strconv.FormatInt(v, 10)
	// 	}
	// 	return ""
	// }

	metadata = make(map[string]string, len(src.Metadata))
	for claim, value := range src.Metadata {
		if vs, ok := value.(string); ok && vs != "" {
			metadata[claim] = vs
			continue
		}
		jsonValue, err := json.Marshal(value)
		if err != nil {
			continue // invalid value
		}
		metadata[claim] = string(jsonValue)
	}

	// extra (identity) claims
	for att, vs := range map[string]string{
		// ".dc":          uint64Value(src.Dc),
		".given_name":  src.GivenName,
		".middle_name": src.MiddleName,
		".family_name": src.FamilyName,
		// ".nickname":           src.Nickname,
		// ".preferred_username": src.PreferredUsername,
		".profile": src.Profile,
		".picture": src.Picture,
		// ".website":            src.Website,
		".email":                 src.Email,
		".email_verified":        boolValue(src.EmailVerified),
		".phone_number":          src.PhoneNumber,
		".phone_number_verified": boolValue(src.PhoneNumberVerified),
		".gender":                src.Gender,
		".birthdate":             src.Birthdate,
		".zoneinfo":              src.Zoneinfo,
		".locale":                src.Locale,
	} {

		if vs == "" {
			continue // ignore empty values
		}
		// populate attribute claim value
		metadata[att] = vs
	}

	if len(metadata) == 0 {
		metadata = nil
	}
	return metadata
}
