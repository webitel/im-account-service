# [TODO]

`[+]` [Re]generate [access_token] for session.(Authorization) on each [Token] request  
`[ ]` Deal with [session].contact reference type for better filter abilities  
`[ ]` TokenRequest: find [X-Webitel-Access] session as hint ; do NOT validate token because it may be outdated ...

## [TODO]: Contacts

```jsonc
// Contact
{
  "id":       "[READONLY]",          // contact id
  "dc":       "[READONLY|REQUIRED]", // business id
  "iss":      "[READONLY|REQUIRED]", // issuer ns
  "sub":      "[READONLY|REQUIRED]", // subject id
  "app":      "[READONLY|OPTIONAL]", // created by
  "type":     "[READONLY|OPTIONAL]", // default: $(iss)
  "name":     "[REQUIRED]",          // common (display) name
  "username": "[OPTIONAL]",          // @mention
  "metadata": "[OPTIONAL]"           // extra claims ; TODO: map[string]any !!!
}
```
```jsonc
// createContact
{
  "dc":       "[REQUIRED]",
  "iss":      "[REQUIRED]",
  "sub":      "[REQUIRED]",
  "app":      "[OPTIONAL]", // FIXME: text gateway(s) has no app assigned
  "type":     "[OPTIONAL]",
  "name":     "[REQUIRED]",
  "username": "[OPTIONAL]",
  "metadata": "[OPTIONAL]"
}
```
```jsonc
// updateContact
{  
  "id":       "[REQUIRED]",
  "name":     "[REQUIRED]",
  "type":     "[OPTIONAL]", // ONLY if not yet defined ; DO NOT reset !
  "username": "[OPTIONAL]",
  "metadata": "[OPTIONAL]"
}
```

## [Text] Gateway(s)

`[ ]` Track account(s) **blocked** status
- Meta  
`[ ]` Separate **App** (Client) configuration from accounts (gates) connected  
`[ ]` How to simplify **Meta App REVIEW** for every On Demand instalation  
- Facebook  
`[ ]` Detect UNIQUE Contact by related **App** [PSID/ASID Matching](https://developers.facebook.com/docs/messenger-platform/identity/id-matching)  