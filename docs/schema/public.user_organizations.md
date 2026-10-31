# public.user_organizations

## Columns

| Name            | Type                     | Default                     | Nullable | Children | Parents                                         | Comment |
| --------------- | ------------------------ | --------------------------- | -------- | -------- | ----------------------------------------------- | ------- |
| user_id         | bigint                   |                             | false    |          | [public.users](public.users.md)                 |         |
| organization_id | bigint                   |                             | false    |          | [public.organizations](public.organizations.md) |         |
| role            | varchar(50)              | 'member'::character varying | false    |          |                                                 |         |
| created_at      | timestamp with time zone | now()                       | false    |          |                                                 |         |
| created_by      | varchar(255)             |                             | true     |          |                                                 |         |

## Constraints

| Name                                        | Type        | Definition                                                                   |
| ------------------------------------------- | ----------- | ---------------------------------------------------------------------------- |
| user_organizations_created_at_not_null      | n           | NOT NULL created_at                                                          |
| user_organizations_organization_id_not_null | n           | NOT NULL organization_id                                                     |
| user_organizations_role_not_null            | n           | NOT NULL role                                                                |
| user_organizations_user_id_not_null         | n           | NOT NULL user_id                                                             |
| user_organizations_organization_id_fkey     | FOREIGN KEY | FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE |
| user_organizations_user_id_fkey             | FOREIGN KEY | FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE                 |
| user_organizations_pkey                     | PRIMARY KEY | PRIMARY KEY (user_id, organization_id)                                       |

## Indexes

| Name                    | Definition                                                                                                      |
| ----------------------- | --------------------------------------------------------------------------------------------------------------- |
| user_organizations_pkey | CREATE UNIQUE INDEX user_organizations_pkey ON public.user_organizations USING btree (user_id, organization_id) |

## Relations

```mermaid
erDiagram

"public.user_organizations" }o--|| "public.users" : "FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE"
"public.child_attendances" }o--o| "public.users" : "FOREIGN KEY (recorded_by) REFERENCES users(id) ON DELETE SET NULL"
"public.government_funding_bill_periods" }o--o| "public.users" : "FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL"
"public.sessions" }o--|| "public.users" : "FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE"
"public.factors" }o--|| "public.users" : "FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE"
"public.user_organizations" }o--|| "public.organizations" : "FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE"
"public.sections" }o--|| "public.organizations" : "FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE"
"public.employees" }o--|| "public.organizations" : "FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE"
"public.pay_plans" }o--|| "public.organizations" : "FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE"
"public.children" }o--|| "public.organizations" : "FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE"
"public.child_attendances" }o--|| "public.organizations" : "FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE"
"public.budget_items" }o--|| "public.organizations" : "FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE"
"public.audit_logs" }o--o| "public.organizations" : "FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE SET NULL"
"public.government_funding_bill_periods" }o--|| "public.organizations" : "FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE"

"public.user_organizations" {
  bigint user_id FK
  bigint organization_id FK
  varchar_50_ role
  timestamp_with_time_zone created_at
  varchar_255_ created_by
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
"public.sections" {
  bigint id
  bigint organization_id FK
  varchar_255_ name
  boolean is_default
  integer min_age_months
  integer max_age_months
  timestamp_with_time_zone created_at
  varchar_255_ created_by
  timestamp_with_time_zone updated_at
  timestamp_with_time_zone deleted_at
}
"public.employees" {
  bigint id
  bigint organization_id FK
  varchar_255_ first_name
  varchar_255_ last_name
  varchar_20_ gender
  date birthdate
  timestamp_with_time_zone created_at
  timestamp_with_time_zone updated_at
}
"public.pay_plans" {
  bigint id
  bigint organization_id FK
  text name
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
"public.budget_items" {
  bigint id
  bigint organization_id FK
  varchar_255_ name
  varchar_50_ category
  boolean per_child
  timestamp_with_time_zone created_at
  timestamp_with_time_zone updated_at
}
"public.audit_logs" {
  bigint id
  timestamp_with_time_zone timestamp
  bigint user_id
  varchar_255_ user_email
  varchar_100_ action
  varchar_100_ resource_type
  bigint resource_id
  varchar_45_ ip_address
  varchar_512_ user_agent
  text details
  boolean success
  bigint organization_id FK
  varchar_64_ request_id
}
```

---

> Generated by [tbls](https://github.com/k1LoW/tbls)
