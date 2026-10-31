# public.child_contracts

## Columns

| Name       | Type                     | Default                                     | Nullable | Children | Parents                               | Comment |
| ---------- | ------------------------ | ------------------------------------------- | -------- | -------- | ------------------------------------- | ------- |
| id         | bigint                   | nextval('child_contracts_id_seq'::regclass) | false    |          |                                       |         |
| child_id   | bigint                   |                                             | false    |          | [public.children](public.children.md) |         |
| from_date  | date                     |                                             | false    |          |                                       |         |
| to_date    | date                     |                                             | true     |          |                                       |         |
| section_id | bigint                   |                                             | false    |          | [public.sections](public.sections.md) |         |
| properties | jsonb                    |                                             | true     |          |                                       |         |
| created_at | timestamp with time zone |                                             | true     |          |                                       |         |
| updated_at | timestamp with time zone |                                             | true     |          |                                       |         |
| version    | bigint                   | 1                                           | false    |          |                                       |         |

## Constraints

| Name                                | Type        | Definition                                                                                                                                              |
| ----------------------------------- | ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| child_contracts_child_id_not_null   | n           | NOT NULL child_id                                                                                                                                       |
| child_contracts_from_date_not_null  | n           | NOT NULL from_date                                                                                                                                      |
| child_contracts_id_not_null         | n           | NOT NULL id                                                                                                                                             |
| child_contracts_section_id_not_null | n           | NOT NULL section_id                                                                                                                                     |
| child_contracts_version_not_null    | n           | NOT NULL version                                                                                                                                        |
| child_contracts_section_id_fkey     | FOREIGN KEY | FOREIGN KEY (section_id) REFERENCES sections(id)                                                                                                        |
| child_contracts_child_id_fkey       | FOREIGN KEY | FOREIGN KEY (child_id) REFERENCES children(id) ON DELETE CASCADE                                                                                        |
| child_contracts_pkey                | PRIMARY KEY | PRIMARY KEY (id)                                                                                                                                        |
| child_contracts_no_overlap          | x           | EXCLUDE USING gist (child_id WITH =, daterange(from_date, COALESCE((to_date + 1), 'infinity'::date), '[)'::text) WITH &&) DEFERRABLE INITIALLY DEFERRED |

## Indexes

| Name                           | Definition                                                                                                                                                           |
| ------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| child_contracts_pkey           | CREATE UNIQUE INDEX child_contracts_pkey ON public.child_contracts USING btree (id)                                                                                  |
| idx_child_contracts_child_id   | CREATE INDEX idx_child_contracts_child_id ON public.child_contracts USING btree (child_id)                                                                           |
| idx_child_contracts_section_id | CREATE INDEX idx_child_contracts_section_id ON public.child_contracts USING btree (section_id)                                                                       |
| idx_child_contracts_period     | CREATE INDEX idx_child_contracts_period ON public.child_contracts USING btree (child_id, from_date, to_date)                                                         |
| child_contracts_no_overlap     | CREATE INDEX child_contracts_no_overlap ON public.child_contracts USING gist (child_id, daterange(from_date, COALESCE((to_date + 1), 'infinity'::date), '[)'::text)) |

## Relations

```mermaid
erDiagram

"public.child_contracts" }o--|| "public.children" : "FOREIGN KEY (child_id) REFERENCES children(id) ON DELETE CASCADE"
"public.child_attendances" }o--|| "public.children" : "FOREIGN KEY (child_id) REFERENCES children(id) ON DELETE CASCADE"
"public.child_vouchers" }o--|| "public.children" : "FOREIGN KEY (child_id) REFERENCES children(id) ON DELETE CASCADE"
"public.children" }o--|| "public.organizations" : "FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE"
"public.child_contracts" }o--|| "public.sections" : "FOREIGN KEY (section_id) REFERENCES sections(id)"
"public.employee_contracts" }o--|| "public.sections" : "FOREIGN KEY (section_id) REFERENCES sections(id)"
"public.sections" }o--|| "public.organizations" : "FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE"

"public.child_contracts" {
  bigint id
  bigint child_id FK
  date from_date
  date to_date
  bigint section_id FK
  jsonb properties
  timestamp_with_time_zone created_at
  timestamp_with_time_zone updated_at
  bigint version
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
"public.child_vouchers" {
  bigint id
  bigint child_id FK
  varchar_17_ voucher_number
  date first_seen
  timestamp_with_time_zone created_at
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
"public.employee_contracts" {
  bigint id
  bigint employee_id FK
  date from_date
  date to_date
  bigint section_id FK
  jsonb properties
  varchar_50_ staff_category
  varchar_20_ grade
  integer step
  double_precision weekly_hours
  bigint pay_plan_id FK
  timestamp_with_time_zone created_at
  timestamp_with_time_zone updated_at
  bigint version
}
```

---

> Generated by [tbls](https://github.com/k1LoW/tbls)
