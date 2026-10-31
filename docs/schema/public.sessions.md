# public.sessions

## Columns

| Name                   | Type                     | Default                      | Nullable | Children | Parents                         | Comment |
| ---------------------- | ------------------------ | ---------------------------- | -------- | -------- | ------------------------------- | ------- |
| id                     | character(64)            |                              | false    |          |                                 |         |
| user_id                | bigint                   |                              | false    |          | [public.users](public.users.md) |         |
| created_at             | timestamp with time zone |                              | false    |          |                                 |         |
| expires_at             | timestamp with time zone |                              | false    |          |                                 |         |
| created_ip             | varchar(45)              |                              | true     |          |                                 |         |
| created_user_agent     | text                     |                              | true     |          |                                 |         |
| kind                   | varchar(32)              | 'regular'::character varying | false    |          |                                 |         |
| mfa_challenge_failures | bigint                   | 0                            | false    |          |                                 |         |
| password_verified_at   | timestamp with time zone |                              | true     |          |                                 |         |
| challenge_nonce        | bytea                    |                              | true     |          |                                 |         |

## Constraints

| Name                                     | Type        | Definition                                                                                                     |
| ---------------------------------------- | ----------- | -------------------------------------------------------------------------------------------------------------- |
| sessions_created_at_not_null             | n           | NOT NULL created_at                                                                                            |
| sessions_expires_at_not_null             | n           | NOT NULL expires_at                                                                                            |
| sessions_id_not_null                     | n           | NOT NULL id                                                                                                    |
| sessions_kind_check                      | CHECK       | CHECK (((kind)::text = ANY ((ARRAY['regular'::character varying, 'pending_mfa'::character varying])::text[]))) |
| sessions_kind_not_null                   | n           | NOT NULL kind                                                                                                  |
| sessions_mfa_challenge_failures_not_null | n           | NOT NULL mfa_challenge_failures                                                                                |
| sessions_user_id_not_null                | n           | NOT NULL user_id                                                                                               |
| sessions_user_id_fkey                    | FOREIGN KEY | FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE                                                   |
| sessions_pkey                            | PRIMARY KEY | PRIMARY KEY (id)                                                                                               |

## Indexes

| Name                      | Definition                                                                               |
| ------------------------- | ---------------------------------------------------------------------------------------- |
| sessions_pkey             | CREATE UNIQUE INDEX sessions_pkey ON public.sessions USING btree (id)                    |
| idx_sessions_user_id      | CREATE INDEX idx_sessions_user_id ON public.sessions USING btree (user_id)               |
| idx_sessions_expires_at   | CREATE INDEX idx_sessions_expires_at ON public.sessions USING btree (expires_at)         |
| idx_sessions_kind_expires | CREATE INDEX idx_sessions_kind_expires ON public.sessions USING btree (kind, expires_at) |

## Relations

```mermaid
erDiagram

"public.sessions" }o--|| "public.users" : "FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE"
"public.user_organizations" }o--|| "public.users" : "FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE"
"public.child_attendances" }o--o| "public.users" : "FOREIGN KEY (recorded_by) REFERENCES users(id) ON DELETE SET NULL"
"public.government_funding_bill_periods" }o--o| "public.users" : "FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL"
"public.factors" }o--|| "public.users" : "FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE"

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
```

---

> Generated by [tbls](https://github.com/k1LoW/tbls)
