# public.factor_webauthn_credentials

## Columns

| Name               | Type                     | Default | Nullable | Children | Parents                             | Comment |
| ------------------ | ------------------------ | ------- | -------- | -------- | ----------------------------------- | ------- |
| factor_id          | bigint                   |         | false    |          | [public.factors](public.factors.md) |         |
| credential_id      | bytea                    |         | false    |          |                                     |         |
| public_key         | bytea                    |         | false    |          |                                     |         |
| aaguid             | bytea                    |         | true     |          |                                     |         |
| sign_count         | bigint                   | 0       | false    |          |                                     |         |
| transports         | varchar(255)             |         | true     |          |                                     |         |
| attestation_format | varchar(64)              |         | true     |          |                                     |         |
| backup_eligible    | boolean                  | false   | false    |          |                                     |         |
| backup_state       | boolean                  | false   | false    |          |                                     |         |
| uv_initialized     | boolean                  | false   | false    |          |                                     |         |
| created_at         | timestamp with time zone |         | false    |          |                                     |         |

## Constraints

| Name                                                 | Type        | Definition                                                       |
| ---------------------------------------------------- | ----------- | ---------------------------------------------------------------- |
| factor_webauthn_credentials_backup_eligible_not_null | n           | NOT NULL backup_eligible                                         |
| factor_webauthn_credentials_backup_state_not_null    | n           | NOT NULL backup_state                                            |
| factor_webauthn_credentials_created_at_not_null      | n           | NOT NULL created_at                                              |
| factor_webauthn_credentials_credential_id_not_null   | n           | NOT NULL credential_id                                           |
| factor_webauthn_credentials_factor_id_not_null       | n           | NOT NULL factor_id                                               |
| factor_webauthn_credentials_public_key_not_null      | n           | NOT NULL public_key                                              |
| factor_webauthn_credentials_sign_count_not_null      | n           | NOT NULL sign_count                                              |
| factor_webauthn_credentials_uv_initialized_not_null  | n           | NOT NULL uv_initialized                                          |
| factor_webauthn_credentials_factor_id_fkey           | FOREIGN KEY | FOREIGN KEY (factor_id) REFERENCES factors(id) ON DELETE CASCADE |
| factor_webauthn_credentials_pkey                     | PRIMARY KEY | PRIMARY KEY (factor_id)                                          |

## Indexes

| Name                              | Definition                                                                                                              |
| --------------------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| factor_webauthn_credentials_pkey  | CREATE UNIQUE INDEX factor_webauthn_credentials_pkey ON public.factor_webauthn_credentials USING btree (factor_id)      |
| idx_factor_webauthn_credential_id | CREATE UNIQUE INDEX idx_factor_webauthn_credential_id ON public.factor_webauthn_credentials USING btree (credential_id) |

## Relations

```mermaid
erDiagram

"public.factor_webauthn_credentials" |o--|| "public.factors" : "FOREIGN KEY (factor_id) REFERENCES factors(id) ON DELETE CASCADE"
"public.factor_totp_secrets" |o--|| "public.factors" : "FOREIGN KEY (factor_id) REFERENCES factors(id) ON DELETE CASCADE"
"public.factor_backup_codes" }o--|| "public.factors" : "FOREIGN KEY (factor_id) REFERENCES factors(id) ON DELETE CASCADE"
"public.factors" }o--|| "public.users" : "FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE"

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
```

---

> Generated by [tbls](https://github.com/k1LoW/tbls)
