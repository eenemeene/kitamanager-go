# public.users

## Columns

| Name          | Type                     | Default                           | Nullable | Children                                                                                                                                                                                                                                                                        | Parents | Comment |
| ------------- | ------------------------ | --------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------- | ------- |
| id            | bigint                   | nextval('users_id_seq'::regclass) | false    | [public.user_organizations](public.user_organizations.md) [public.child_attendances](public.child_attendances.md) [public.government_funding_bill_periods](public.government_funding_bill_periods.md) [public.sessions](public.sessions.md) [public.factors](public.factors.md) |         |         |
| name          | varchar(255)             |                                   | false    |                                                                                                                                                                                                                                                                                 |         |         |
| email         | varchar(255)             |                                   | false    |                                                                                                                                                                                                                                                                                 |         |         |
| password      | varchar(255)             |                                   | false    |                                                                                                                                                                                                                                                                                 |         |         |
| active        | boolean                  | true                              | true     |                                                                                                                                                                                                                                                                                 |         |         |
| is_superadmin | boolean                  | false                             | true     |                                                                                                                                                                                                                                                                                 |         |         |
| last_login    | timestamp with time zone |                                   | true     |                                                                                                                                                                                                                                                                                 |         |         |
| created_at    | timestamp with time zone |                                   | true     |                                                                                                                                                                                                                                                                                 |         |         |
| created_by    | varchar(255)             |                                   | true     |                                                                                                                                                                                                                                                                                 |         |         |
| updated_at    | timestamp with time zone |                                   | true     |                                                                                                                                                                                                                                                                                 |         |         |
| deleted_at    | timestamp with time zone |                                   | true     |                                                                                                                                                                                                                                                                                 |         |         |

## Constraints

| Name                    | Type        | Definition        |
| ----------------------- | ----------- | ----------------- |
| users_email_not_null    | n           | NOT NULL email    |
| users_id_not_null       | n           | NOT NULL id       |
| users_name_not_null     | n           | NOT NULL name     |
| users_password_not_null | n           | NOT NULL password |
| users_pkey              | PRIMARY KEY | PRIMARY KEY (id)  |

## Indexes

| Name                  | Definition                                                                                                              |
| --------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| users_pkey            | CREATE UNIQUE INDEX users_pkey ON public.users USING btree (id)                                                         |
| idx_users_deleted_at  | CREATE INDEX idx_users_deleted_at ON public.users USING btree (deleted_at) WHERE (deleted_at IS NOT NULL)               |
| idx_users_email_lower | CREATE UNIQUE INDEX idx_users_email_lower ON public.users USING btree (lower((email)::text)) WHERE (deleted_at IS NULL) |

## Relations

```mermaid
erDiagram

"public.user_organizations" }o--|| "public.users" : "FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE"
"public.user_organizations" }o--|| "public.organizations" : "FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE"
"public.child_attendances" }o--o| "public.users" : "FOREIGN KEY (recorded_by) REFERENCES users(id) ON DELETE SET NULL"
"public.child_attendances" }o--|| "public.children" : "FOREIGN KEY (child_id) REFERENCES children(id) ON DELETE CASCADE"
"public.child_attendances" }o--|| "public.organizations" : "FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE"
"public.government_funding_bill_periods" }o--o| "public.users" : "FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL"
"public.government_funding_bill_children" }o--|| "public.government_funding_bill_periods" : "FOREIGN KEY (period_id) REFERENCES government_funding_bill_periods(id) ON DELETE CASCADE"
"public.government_funding_bill_periods" }o--|| "public.organizations" : "FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE"
"public.sessions" }o--|| "public.users" : "FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE"
"public.factors" }o--|| "public.users" : "FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE"
"public.factor_totp_secrets" |o--|| "public.factors" : "FOREIGN KEY (factor_id) REFERENCES factors(id) ON DELETE CASCADE"
"public.factor_backup_codes" }o--|| "public.factors" : "FOREIGN KEY (factor_id) REFERENCES factors(id) ON DELETE CASCADE"
"public.factor_webauthn_credentials" |o--|| "public.factors" : "FOREIGN KEY (factor_id) REFERENCES factors(id) ON DELETE CASCADE"

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
"public.organizations" {
  bigint id
  varchar_255_ name
  boolean active
  varchar_50_ state
  timestamp_with_time_zone created_at
  varchar_255_ created_by
  timestamp_with_time_zone updated_at
  timestamp_with_time_zone deleted_at
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
"public.children" {
  bigint id
  bigint organization_id FK
  varchar_255_ first_name
  varchar_255_ last_name
  varchar_20_ gender
  date birthdate
  timestamp_with_time_zone created_at
  timestamp_with_time_zone updated_at
  date school_entry_date
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
"public.government_funding_bill_children" {
  bigint id
  bigint period_id FK
  varchar_20_ voucher_number
  varchar_255_ child_name
  varchar_10_ birth_date
  integer district
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
```

---

> Generated by [tbls](https://github.com/k1LoW/tbls)
