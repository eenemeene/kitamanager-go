---
title: Errors
weight: 3
---

Every API error is an [RFC 9457](https://www.rfc-editor.org/rfc/rfc9457) problem
document, sent as `Content-Type: application/problem+json`. That includes the
responses the router produces itself — an unrouted path, a wrong method, a panic
— so a client can parse one shape for every failure and never has to handle a
plain-text body.

```json
{
  "type": "https://eenemeene.github.io/kitamanager-go/en/docs/reference/api/errors/#contract_overlap",
  "title": "Contract periods overlap",
  "status": 409,
  "detail": "contract periods overlap between 2026-01-01 and 2026-03-31",
  "instance": "/api/v1/organizations/1/children/42/contracts/7/amend",
  "code": "contract_overlap",
  "request_id": "0e03dc7d-9baa-4a23-a8ba-bc54ad5b30b9"
}
```

## Language

The API is English. `code`, `type`, field names and every log line stay English
regardless of what a client asks for, and English is what you get by default.

`Accept-Language` negotiates the prose members. Send `de` and `title` comes back
in German; the response states what it actually served in `Content-Language`, and
carries `Vary: Accept-Language` so a shared cache keys on it. Quality values and
regional subtags are honoured — `de-AT;q=0.9, en;q=0.8` selects German — and an
unsupported or malformed value falls back to English rather than failing.

`detail` is still English in every language today: it is composed per occurrence
at the point the error is raised, and those sites have not been converted yet. A
localized client should keep translating `code` for now.

## Which member to use

| Member | Use it for |
|---|---|
| `code` | **Branching in code.** The stable identifier; it does not change when a message is reworded. |
| `type` | A link to this page, anchored at the code. Same information as `code`, in the form the specification defines. |
| `title` | A short summary of the problem type. Identical for every occurrence of a code. |
| `detail` | What happened *this* time — which contract, which dates, which field. |
| `instance` | The request path. |
| `request_id` | Quote this when reporting a problem; it matches the server log line. |
| `invalid_params` | Present on validation failures: one entry per rejected field. |
| `params` | The specifics as key/value data, where the endpoint provides them. |

`title` and `detail` are English and always will be — the API has one language.
Anything user-facing translates `code` on its side; that is what the KitaManager
frontend does, which is why a German user sees German for a failure whose
`detail` is an English sentence.

A translated sentence is necessarily more generic than a `detail` written for one
occurrence, so translating must not cost the reader the specifics. Two members
exist for that: `params` carries the data behind the sentence, for a client to
interpolate into its own wording, and `invalid_params` carries `rule` and `param`
alongside the English `reason` so a rejected field can be described in any
language. Where neither is available, a localized client should show `detail` as
well as its translation rather than instead of it.

## Validation failures

A 400 with `code: validation_error` names the fields it rejected, so a form can
mark the offending inputs instead of printing one sentence above all of them.

```json
{
  "type": "https://eenemeene.github.io/kitamanager-go/en/docs/reference/api/errors/#validation_error",
  "title": "Validation failed",
  "status": 400,
  "detail": "email must be a valid email address; weekly_hours is required",
  "code": "validation_error",
  "invalid_params": [
    { "field": "email", "reason": "must be a valid email address", "rule": "email" },
    { "field": "weekly_hours", "reason": "is required", "rule": "required" }
  ]
}
```

`field` is the JSON path within the request body, so nested fields read as
`properties.0.name`. `rule` is the validator tag that rejected it — `required`,
`email`, `min`, `max`, `voucher` — and `param` its argument where the rule takes
one, so a client can build the sentence itself instead of showing `reason`.

## Codes

### not_found

**404.** The addressed resource does not exist, or is outside the organizations
this session may see — the two are deliberately indistinguishable, so probing an
id tells an attacker nothing.

### bad_request

**400.** The request could not be understood: malformed JSON, an unknown field
(bodies reject fields they do not declare), more than one JSON value, or a body
over the size limit.

### validation_error

**400.** The body parsed but failed its field rules. See `invalid_params` above.

### conflict

**409.** The request contradicts the current state of the resource.

### contract_overlap

**409.** The contract periods would cover the same day twice. Adjust the dates so
the periods are adjacent rather than overlapping, or use the boundary endpoint,
which moves the seam between two periods in one call and cannot produce an
overlap by construction.

### email_conflict

**409.** Another user already has that email address.

### duplicate_bill_hash

**409.** This bill file has already been imported. The check is on the file's
content hash, so re-uploading a renamed copy is still caught.

### duplicate_bill_month

**409.** A bill already exists for that month.

### unauthorized

**401.** No session, an expired session, or credentials that did not verify.
Clients should clear local session state and send the user to sign in — with one
exception: a 401 from `/login`, `/logout` or the MFA endpoints is part of the
normal flow of those endpoints and does not mean the existing session went stale.

### forbidden

**403.** Authenticated, but not permitted: the role lacks the permission, or the
request reaches outside the session's organizations. Also returned by the CSRF
check when a cookie-authenticated unsafe request arrives without a matching
`X-CSRF-Token`.

### precondition_required

**428.** A write that requires optimistic concurrency control arrived without an
`If-Match` header. Read the resource, take its `ETag` (or the `version` on the
body), and send it back. See [contract writes](../#contract-writes-worked-examples).

### precondition_failed

**412.** The `If-Match` version is not the current one: someone else changed the
resource since it was read. Re-read it, show the user what changed, and apply the
edit again. This is distinct from `contract_overlap` — one is a stale read, the
other is an invalid date range — so a client can tell "reload and retry" from
"fix the dates".

### too_many_requests

**429.** Rate limited. The `Retry-After` header carries the wait in seconds.

### method_not_allowed

**405.** The path exists, the method does not.

### internal_error

**500.** Something failed on the server. `detail` is deliberately fixed text —
the underlying error can carry a query fragment or a driver message, and that
belongs in the log, not in a response. `request_id` is how the two are matched
up; quote it in a bug report.
