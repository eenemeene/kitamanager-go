# public.government_funding_bill_children

## Description

## Columns

| Name           | Type         | Default                                                      | Nullable | Children                                                                              | Parents                                                                             | Comment |
| -------------- | ------------ | ------------------------------------------------------------ | -------- | ------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------- | ------- |
| id             | bigint       | nextval('government_funding_bill_children_id_seq'::regclass) | false    | [public.government_funding_bill_payments](public.government_funding_bill_payments.md) |                                                                                     |         |
| period_id      | bigint       |                                                              | false    |                                                                                       | [public.government_funding_bill_periods](public.government_funding_bill_periods.md) |         |
| voucher_number | varchar(20)  |                                                              | false    |                                                                                       |                                                                                     |         |
| child_name     | varchar(255) |                                                              | false    |                                                                                       |                                                                                     |         |
| birth_date     | varchar(10)  |                                                              | false    |                                                                                       |                                                                                     |         |
| district       | integer      |                                                              | false    |                                                                                       |                                                                                     |         |

## Constraints

| Name                                                     | Type        | Definition                                                                               |
| -------------------------------------------------------- | ----------- | ---------------------------------------------------------------------------------------- |
| government_funding_bill_children_birth_date_not_null     | n           | NOT NULL birth_date                                                                      |
| government_funding_bill_children_child_name_not_null     | n           | NOT NULL child_name                                                                      |
| government_funding_bill_children_district_not_null       | n           | NOT NULL district                                                                        |
| government_funding_bill_children_id_not_null             | n           | NOT NULL id                                                                              |
| government_funding_bill_children_period_id_not_null      | n           | NOT NULL period_id                                                                       |
| government_funding_bill_children_voucher_number_not_null | n           | NOT NULL voucher_number                                                                  |
| government_funding_bill_children_period_id_fkey          | FOREIGN KEY | FOREIGN KEY (period_id) REFERENCES government_funding_bill_periods(id) ON DELETE CASCADE |
| government_funding_bill_children_pkey                    | PRIMARY KEY | PRIMARY KEY (id)                                                                         |

## Indexes

| Name                                  | Definition                                                                                                            |
| ------------------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| government_funding_bill_children_pkey | CREATE UNIQUE INDEX government_funding_bill_children_pkey ON public.government_funding_bill_children USING btree (id) |
| idx_gfbc_period                       | CREATE INDEX idx_gfbc_period ON public.government_funding_bill_children USING btree (period_id)                       |

## Relations

```mermaid
erDiagram

"public.government_funding_bill_payments" }o--|| "public.government_funding_bill_children" : "FOREIGN KEY (child_id) REFERENCES government_funding_bill_children(id) ON DELETE CASCADE"
"public.government_funding_bill_children" }o--|| "public.government_funding_bill_periods" : "FOREIGN KEY (period_id) REFERENCES government_funding_bill_periods(id) ON DELETE CASCADE"
"public.government_funding_bill_periods" }o--|| "public.organizations" : "FOREIGN KEY (organization_id) REFERENCES organizations(id)"
"public.government_funding_bill_periods" }o--|| "public.users" : "FOREIGN KEY (created_by) REFERENCES users(id)"

"public.government_funding_bill_children" {
  bigint id
  bigint period_id FK
  varchar_20_ voucher_number
  varchar_255_ child_name
  varchar_10_ birth_date
  integer district
}
"public.government_funding_bill_payments" {
  bigint id
  bigint child_id FK
  varchar_100_ key
  varchar_255_ value
  integer amount
  bigint row_index
  varchar_20_ row_type
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
}
"public.organizations" {
  bigint id
  varchar_255_ name
  boolean active
  varchar_50_ state
  timestamp_with_time_zone created_at
  varchar_255_ created_by
  timestamp_with_time_zone updated_at
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
}
```

---

> Generated by [tbls](https://github.com/k1LoW/tbls)
