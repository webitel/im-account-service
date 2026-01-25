package model

import "time"

// Contact reference
type ContactId struct {
	Dc  int64  // Business Account ID
	Id  string // Contact Internal ID
	Iss string // Issuer identifier ; namespace
	Sub string // Subject identifier, under Issuer
}

// Contact profile
type Contact struct {
	// [IM] Business [Domain] Account ID
	Dc int64
	// [IM] Service [Provider] Account ID ; issued for UNIQUE( Dc + Iss + Sub )
	Id string
	// [IM] Application ID this Contact was registered with
	App string

	// ------------------------------------ //
	//           Identity Claims            //
	// ------------------------------------ //

	// REQUIRED. Issuer Identifier for the Issuer of the response.
	// The iss value is a case sensitive URL using the https scheme that contains scheme, host,
	// and optionally, port number and path components and no query or fragment components.
	Iss string
	// REQUIRED. Subject Identifier.
	// A locally unique and never reassigned identifier within the Issuer for the End-User,
	// which is intended to be consumed by the Client, e.g., 24400320 or AItOawmwtWwcT0k51BayewNvutrJUqsvl6qs7A4.
	// It MUST NOT exceed 255 ASCII characters in length.
	// The sub value is a case sensitive string.
	Sub string
	// Well-known Contact(s) Issuer [proto]col type ; app registered
	// Default: [Iss]
	Type string
	// REQUIRED. End-User's full name in displayable form including all name parts,
	// possibly including titles and suffixes, ordered according to the End-User's locale and preferences.
	Name string
	// Preferred Username
	Username string
	// Given name(s) or first name(s) of the End-User.
	// Note that in some cultures, people can have multiple given names;
	// all can be present, with the names being separated by space characters.
	GivenName string
	// Middle name(s) of the End-User.
	// Note that in some cultures, people can have multiple middle names;
	// all can be present, with the names being separated by space characters.
	// Also note that in some cultures, middle names are not used.
	MiddleName string
	// Surname(s) or last name(s) of the End-User.
	// Note that in some cultures, people can have multiple family names or no family name;
	// all can be present, with the names being separated by space characters.
	FamilyName string
	// OPTIONAL. End-User's birthday, represented as an ISO 8601:2004 [ISO8601‑2004] YYYY-MM-DD format.
	// The year MAY be 0000, indicating that it is omitted.
	// To represent only the year, YYYY format is allowed.
	Birthdate string
	// OPTIONAL. String from zoneinfo [zoneinfo] time zone database representing the End-User's time zone.
	// For example, Europe/Kyiv or America/Los_Angeles.
	Zoneinfo string
	// OPTIONAL. URL of the End-User's profile page.
	// The contents of this Web page SHOULD be about the End-User.
	// NOTE: Issuer SP (IdP) related URL.
	Profile string
	// OPTIONAL. URL of the End-User's profile picture.
	// This URL MUST refer to an image file
	// (for example, a PNG, JPEG, or GIF image file),
	// rather than to a Web page containing an image.
	Picture string
	// OPTIONAL. End-User's gender.
	// Values defined by this specification are `female` and `male`.
	// Other values MAY be used when neither of the defined values are applicable.
	Gender string
	// End-User's locale, represented as a BCP47 [RFC5646] language tag.
	// This is typically an ISO 639-1 Alpha-2 [ISO639‑1] language code in lowercase
	// and an ISO 3166-1 Alpha-2 [ISO3166‑1] country code in uppercase,
	// separated by a dash. For example, `en-US` or `uk-UA`.
	Locale string
	// End-User's preferred e-mail address.
	// Its value MUST conform to the RFC 5322 [RFC5322] addr-spec syntax.
	// The RP MUST NOT rely upon this value being unique, as discussed in Section 5.7.
	Email string
	// True if the End-User's e-mail address has been verified; otherwise false.
	EmailVerified bool
	// End-User's preferred telephone number.
	// E.164 is RECOMMENDED as the format of this Claim, for example, +1 (425) 555-1212 or +56 (2) 687 2400.
	// If the phone number contains an extension, it is RECOMMENDED that
	// the extension be represented using the RFC 3966 [RFC3966] extension syntax, for example, +1 (604) 555-1234;ext=5678.
	PhoneNumber string
	// True if the End-User's phone number has been verified; otherwise false.
	PhoneNumberVerified bool
	// End-User's extra attributes (claims) metadata.
	Metadata map[string]any // *structpb.Struct
	// Time the End-User's information was last updated.
	// Its value is a JSON number representing the number of seconds from 1970-01-01T0:0:0Z as measured in UTC until the date/time.
	CreatedAt time.Time  // int64
	UpdatedAt *time.Time // int64
	DeletedAt *time.Time // int64
}
