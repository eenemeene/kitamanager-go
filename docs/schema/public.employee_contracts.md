# public.employee_contracts

## Columns

| Name           | Type                     | Default                                        | Nullable | Children | Parents                                 | Comment |
| -------------- | ------------------------ | ---------------------------------------------- | -------- | -------- | --------------------------------------- | ------- |
| id             | bigint                   | nextval('employee_contracts_id_seq'::regclass) | false    |          |                                         |         |
| employee_id    | bigint                   |                                                | false    |          | [public.employees](public.employees.md) |         |
| from_date      | date                     |                                                | false    |          |                                         |         |
| to_date        | date                     |                                                | true     |          |                                         |         |
| section_id     | bigint                   |                                                | false    |          | [public.sections](public.sections.md)   |         |
| properties     | jsonb                    |                                                | true     |          |                                         |         |
| staff_category | varchar(50)              | 'qualified'::character varying                 | false    |          |                                         |         |
| grade          | varchar(20)              |                                                | true     |          |                                         |         |
| step           | integer                  |                                                | true     |          |                                         |         |
| weekly_hours   | double precision         |                                                | true     |          |                                         |         |
| pay_plan_id    | bigint                   |                                                | false    |          | [public.pay_plans](public.pay_plans.md) |         |
| created_at     | timestamp with time zone |                                                | true     |          |                                         |         |
| updated_at     | timestamp with time zone |                                                | true     |          |                                         |         |
| version        | bigint                   | 1                                              | false    |          |                                         |         |

## Constraints

| Name                                       | Type        | Definition                                                                                                                                                         |
| ------------------------------------------ | ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| employee_contracts_employee_id_not_null    | n           | NOT NULL employee_id                                                                                                                                               |
| employee_contracts_from_date_not_null      | n           | NOT NULL from_date                                                                                                                                                 |
| employee_contracts_id_not_null             | n           | NOT NULL id                                                                                                                                                        |
| employee_contracts_pay_plan_id_not_null    | n           | NOT NULL pay_plan_id                                                                                                                                               |
| employee_contracts_section_id_not_null     | n           | NOT NULL section_id                                                                                                                                                |
| employee_contracts_staff_category_not_null | n           | NOT NULL staff_category                                                                                                                                            |
| employee_contracts_staff_category_valid    | CHECK       | CHECK (((staff_category)::text = ANY ((ARRAY['qualified'::character varying, 'supplementary'::character varying, 'non_pedagogical'::character varying])::text[]))) |
| employee_contracts_step_valid              | CHECK       | CHECK (((step IS NULL) OR ((step >= 0) AND (step <= 10))))                                                                                                         |
| employee_contracts_version_not_null        | n           | NOT NULL version                                                                                                                                                   |
| employee_contracts_weekly_hours_valid      | CHECK       | CHECK (((weekly_hours IS NULL) OR ((weekly_hours >= (0)::double precision) AND (weekly_hours <= (168)::double precision))))                                        |
| employee_contracts_section_id_fkey         | FOREIGN KEY | FOREIGN KEY (section_id) REFERENCES sections(id)                                                                                                                   |
| employee_contracts_employee_id_fkey        | FOREIGN KEY | FOREIGN KEY (employee_id) REFERENCES employees(id) ON DELETE CASCADE                                                                                               |
| employee_contracts_pay_plan_id_fkey        | FOREIGN KEY | FOREIGN KEY (pay_plan_id) REFERENCES pay_plans(id)                                                                                                                 |
| employee_contracts_pkey                    | PRIMARY KEY | PRIMARY KEY (id)                                                                                                                                                   |
| employee_contracts_no_overlap              | x           | EXCLUDE USING gist (employee_id WITH =, daterange(from_date, COALESCE((to_date + 1), 'infinity'::date), '[)'::text) WITH &&) DEFERRABLE INITIALLY DEFERRED         |

## Indexes

| Name                               | Definition                                                                                                                                                                    |
| ---------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| employee_contracts_pkey            | CREATE UNIQUE INDEX employee_contracts_pkey ON public.employee_contracts USING btree (id)                                                                                     |
| idx_employee_contracts_employee_id | CREATE INDEX idx_employee_contracts_employee_id ON public.employee_contracts USING btree (employee_id)                                                                        |
| idx_employee_contracts_section_id  | CREATE INDEX idx_employee_contracts_section_id ON public.employee_contracts USING btree (section_id)                                                                          |
| idx_employee_contracts_pay_plan_id | CREATE INDEX idx_employee_contracts_pay_plan_id ON public.employee_contracts USING btree (pay_plan_id)                                                                        |
| idx_employee_contracts_period      | CREATE INDEX idx_employee_contracts_period ON public.employee_contracts USING btree (employee_id, from_date, to_date)                                                         |
| employee_contracts_no_overlap      | CREATE INDEX employee_contracts_no_overlap ON public.employee_contracts USING gist (employee_id, daterange(from_date, COALESCE((to_date + 1), 'infinity'::date), '[)'::text)) |

## Relations

```mermaid
erDiagram

"public.employee_contracts" }o--|| "public.employees" : "FOREIGN KEY (employee_id) REFERENCES employees(id) ON DELETE CASCADE"
"public.employees" }o--|| "public.organizations" : "FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE"
"public.employee_contracts" }o--|| "public.sections" : "FOREIGN KEY (section_id) REFERENCES sections(id)"
"public.child_contracts" }o--|| "public.sections" : "FOREIGN KEY (section_id) REFERENCES sections(id)"
"public.sections" }o--|| "public.organizations" : "FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE"
"public.employee_contracts" }o--|| "public.pay_plans" : "FOREIGN KEY (pay_plan_id) REFERENCES pay_plans(id)"
"public.pay_plan_periods" }o--|| "public.pay_plans" : "FOREIGN KEY (pay_plan_id) REFERENCES pay_plans(id) ON DELETE CASCADE"
"public.pay_plans" }o--|| "public.organizations" : "FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE"

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
"public.pay_plans" {
  bigint id
  bigint organization_id FK
  text name
  timestamp_with_time_zone created_at
  timestamp_with_time_zone updated_at
}
"public.pay_plan_periods" {
  bigint id
  bigint pay_plan_id FK
  date from_date
  date to_date
  double_precision weekly_hours
  integer employer_contribution_rate
  timestamp_with_time_zone created_at
  timestamp_with_time_zone updated_at
}
```

---

> Generated by [tbls](https://github.com/k1LoW/tbls)
