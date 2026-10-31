# public.factors

## Columns

| Name                              | Type                     | Default                             | Nullable | Children                                                                                                                                                                                            | Parents                         | Comment |
| --------------------------------- | ------------------------ | ----------------------------------- | -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------- | ------- |
| id                                | bigint                   | nextval('factors_id_seq'::regclass) | false    | [public.factor_totp_secrets](public.factor_totp_secrets.md) [public.factor_backup_codes](public.factor_backup_codes.md) [public.factor_webauthn_credentials](public.factor_webauthn_credentials.md) |                                 |         |
| user_id                           | bigint                   |                                     | false    |                                                                                                                                                                                                     | [public.users](public.users.md) |         |
| type                              | varchar(32)              |                                     | false    |                                                                                                                                                                                                     |                                 |         |
| label                             | varchar(100)             |                                     | true     |                                                                                                                                                                                                     |                                 |         |
| enabled_at                        | timestamp with time zone |                                     | true     |                                                                                                                                                                                                     |                                 |         |
| last_used_at                      | timestamp with time zone |                                     | true     |                                                                                                                                                                                                     |                                 |         |
| activation_failures               | bigint                   | 0                                   | false    |                                                                                                                                                                                                     |                                 |         |
| created_at                        | timestamp with time zone |                                     | false    |                                                                                                                                                                                                     |                                 |         |
| registration_challenge            | bytea                    |                                     | true     |                                                                                                                                                                                                     |                                 |         |
| registration_challenge_expires_at | timestamp with time zone |                                     | true     |                                                                                                                                                                                                     |                                 |         |

## Constraints

| Name                                 | Type        | Definition                                                                                                                                  |
| ------------------------------------ | ----------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| factors_activation_failures_not_null | n           | NOT NULL activation_failures                                                                                                                |
| factors_created_at_not_null          | n           | NOT NULL created_at                                                                                                                         |
| factors_id_not_null                  | n           | NOT NULL id                                                                                                                                 |
| factors_type_check                   | CHECK       | CHECK (((type)::text = ANY ((ARRAY['totp'::character varying, 'backup_codes'::character varying, 'webauthn'::character varying])::text[]))) |
| factors_type_not_null                | n           | NOT NULL type                                                                                                                               |
| factors_user_id_not_null             | n           | NOT NULL user_id                                                                                                                            |
| factors_user_id_fkey                 | FOREIGN KEY | FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE                                                                                |
| factors_pkey                         | PRIMARY KEY | PRIMARY KEY (id)                                                                                                                            |

## Indexes

| Name                              | Definition                                                                                                                                |
| --------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| factors_pkey                      | CREATE UNIQUE INDEX factors_pkey ON public.factors USING btree (id)                                                                       |
| idx_factors_user_id               | CREATE INDEX idx_factors_user_id ON public.factors USING btree (user_id)                                                                  |
| idx_factors_user_enabled          | CREATE INDEX idx_factors_user_enabled ON public.factors USING btree (user_id, enabled_at) WHERE (enabled_at IS NOT NULL)                  |
| idx_factors_user_singleton_backup | CREATE UNIQUE INDEX idx_factors_user_singleton_backup ON public.factors USING btree (user_id) WHERE ((type)::text = 'backup_codes'::text) |

## Relations

```mermaid
erDiagram

"public.factor_totp_secrets" |o--|| "public.factors" : "FOREIGN KEY (factor_id) REFERENCES factors(id) ON DELETE CASCADE"
"public.factor_backup_codes" }o--|| "public.factors" : "FOREIGN KEY (factor_id) REFERENCES factors(id) ON DELETE CASCADE"
"public.factor_webauthn_credentials" |o--|| "public.factors" : "FOREIGN KEY (factor_id) REFERENCES factors(id) ON DELETE CASCADE"
"public.factors" }o--|| "public.users" : "FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE"
"public.user_organizations" }o--|| "public.users" : "FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE"
"public.child_attendances" }o--o| "public.users" : "FOREIGN KEY (recorded_by) REFERENCES users(id) ON DELETE SET NULL"
"public.government_funding_bill_periods" }o--o| "public.users" : "FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL"
"public.sessions" }o--|| "public.users" : "FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE"

"public.factors" {
  bigint id
  bigint user_id FK
  varchar_32_ type
  varchar_100_ label
  timestamp_with_time_zone enabled_at
  timestamp_with_time_zone last_used_at
  bigint activation_failures
  timestamp_with_time_zone created_at
  bytea registration_challenge
  timestamp_with_time_zone registration_challenge_expires_at
}
"public.factor_totp_secrets" {
  bigint factor_id FK
  bytea secret_ciphertext
  bytea secret_nonce
  bigint last_used_step
}
"public.factor_backup_codes" {
  bigint id
  bigint factor_id FK
  text code_hash
  timestamp_with_time_zone used_at
  timestamp_with_time_zone created_at
}
"public.factor_webauthn_credentials" {
  bigint factor_id FK
  bytea credential_id
  bytea public_key
  bytea aaguid
  bigint sign_count
  varchar_255_ transports
  varchar_64_ attestation_format
  boolean backup_eligible
  boolean backup_state
  boolean uv_initialized
  timestamp_with_time_zone created_at
}
"public.users" {
  bigint id
  varchar_255_ name
  varchar_255_ email
  varchar_255_ password
  boolean active
  boolean is_superadmin
  timestamp_with_time_zone last_login
  timestamp_with_time_zone created_at
  varchar_255_ created_by
  timestamp_with_time_zone updated_at
  timestamp_with_time_zone deleted_at
}
"public.user_organizations" {
  bigint user_id FK
  bigint organization_id FK
  varchar_50_ role
  timestamp_with_time_zone created_at
  varchar_255_ created_by
}
"public.child_attendances" {
  bigint id
  bigint child_id FK
  bigint organization_id FK
  date date
  timestamp_with_time_zone check_in_time
  timestamp_with_time_zone check_out_time
  varchar_20_ status
  varchar_500_ note
  bigint recorded_by FK
  timestamp_with_time_zone created_at
  timestamp_with_time_zone updated_at
}
"public.government_funding_bill_periods" {
  bigint id
  bigint organization_id FK
  date from_date
  date to_date
  varchar_255_ file_name
  varchar_64_ file_sha256
  varchar_255_ facility_name
  integer facility_total
  integer contract_booking
  integer correction_booking
  bigint created_by FK
  timestamp_with_time_zone created_at
  timestamp_with_time_zone updated_at
  date correction_booking_from_month
  date correction_booking_to_month
  varchar_64_ correction_booking_reason
}
"public.sessions" {
  character_64_ id
  bigint user_id FK
  timestamp_with_time_zone created_at
  timestamp_with_time_zone expires_at
  varchar_45_ created_ip
  text created_user_agent
  varchar_32_ kind
  bigint mfa_challenge_failures
  timestamp_with_time_zone password_verified_at
  bytea challenge_nonce
}
```

---

> Generated by [tbls](https://github.com/k1LoW/tbls)
