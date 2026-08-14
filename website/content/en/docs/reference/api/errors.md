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

The API is English, and the top level of a problem document always is: `code`,
`type`, `title`, `detail`, field names and every log line. That is deliberate —
a captured error response stays readable by whoever picks up the support ticket,
whatever language the user was working in.

Send `Accept-Language: de` and the document gains a `localized` member carrying
the same title and detail in German. Nothing above it changes.

```json
{
  "type": "https://eenemeene.github.io/kitamanager-go/en/docs/reference/api/errors/#not_found",
  "title": "Resource not found",
  "status": 404,
  "detail": "child 7 not found in this organization",
  "code": "not_found",
  "request_id": "0e03dc7d-9baa-4a23-a8ba-bc54ad5b30b9",
  "localized": {
    "locale": "de",
    "title": "Ressource nicht gefunden",
    "detail": "Kind 7 wurde in dieser Organisation nicht gefunden"
  }
}
```

The rule for a client: **branch on `code`, show `localized.detail` when present
and `detail` otherwise, log `detail`.**

`localized` is omitted entirely for an English request — the top level already is
that language. `locale` states what was actually served, which is not always what
was asked for: an unsupported language is answered in English. Quality values and
regional subtags are honoured, so `de-AT;q=0.9, en;q=0.8` selects German.

Because the body then carries two languages, `Content-Language` lists both
(`en, de`) rather than just the negotiated one. `Vary: Accept-Language` is always
set so a shared cache keys correctly.

Every user-facing message is translated, and a test fails the build if one is
added or reworded without a translation — so a missing `localized` means English
was requested, not that a translation was forgotten.

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
| `localized` | The same title and detail in the negotiated language. Absent for English. |
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
    { "field": "email", "reason": "must be a valid email address", "rule": "email",
      "localized_reason": "muss eine gültige E-Mail-Adresse sein" },
    { "field": "weekly_hours", "reason": "is required", "rule": "required",
      "localized_reason": "ist erforderlich" }
  ]
}
```

`field` is the JSON path within the request body, so nested fields read as
`properties.0.name` and a bulk import reads as
`add_children[3].contracts[1].from`.

The path is never part of the sentence. It used to be — a bulk import answered
`add_children[3].contracts[1]: from is required` — which forced a client to parse
prose to learn which field failed, and made the German version half German and
half JSON path. Every specification in this space keeps the two apart, and so
does this: locate the field with `field`, show the reason from `reason` or
`localized_reason`, or ignore both and render your own text from `rule`.

`rule` is drawn from a small vocabulary so it can be rendered in any language:
`required`, `non_empty`, `min` (string length), `min_value` (numeric), `max`,
`positive`, `mismatch`, `email`, `voucher`.

Several fields can fail at once, and all of them are reported: the binding
validator returns every failing field rather than the first, and a bulk import
reports every bad row. `rule` is the validator tag that rejected it — `required`,
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
