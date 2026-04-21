# public.government_funding_periods

## Description

## Columns

| Name                   | Type                     | Default                                                | Nullable | Children                                                                        | Parents                                                     | Comment |
| ---------------------- | ------------------------ | ------------------------------------------------------ | -------- | ------------------------------------------------------------------------------- | ----------------------------------------------------------- | ------- |
| id                     | bigint                   | nextval('government_funding_periods_id_seq'::regclass) | false    | [public.government_funding_properties](public.government_funding_properties.md) |                                                             |         |
| government_funding_id  | bigint                   |                                                        | false    |                                                                                 | [public.government_fundings](public.government_fundings.md) |         |
| from_date              | date                     |                                                        | false    |                                                                                 |                                                             |         |
| to_date                | date                     |                                                        | true     |                                                                                 |                                                             |         |
| full_time_weekly_hours | double precision         |                                                        | false    |                                                                                 |                                                             |         |
| comment                | varchar(1000)            |                                                        | true     |                                                                                 |                                                             |         |
| created_at             | timestamp with time zone |                                                        | true     |                                                                                 |                                                             |         |
| updated_at             | timestamp with time zone |                                                        | true     |                                                                                 |                                                             |         |

## Constraints

| Name                                                       | Type        | Definition                                                                               |
| ---------------------------------------------------------- | ----------- | ---------------------------------------------------------------------------------------- |
| government_funding_periods_from_date_not_null              | n           | NOT NULL from_date                                                                       |
| government_funding_periods_full_time_weekly_hours_not_null | n           | NOT NULL full_time_weekly_hours                                                          |
| government_funding_periods_government_funding_id_not_null  | n           | NOT NULL government_funding_id                                                           |
| government_funding_periods_id_not_null                     | n           | NOT NULL id                                                                              |
| government_funding_periods_government_funding_id_fkey      | FOREIGN KEY | FOREIGN KEY (government_funding_id) REFERENCES government_fundings(id) ON DELETE CASCADE |
| government_funding_periods_pkey                            | PRIMARY KEY | PRIMARY KEY (id)                                                                         |

## Indexes

| Name                               | Definition                                                                                                                      |
| ---------------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| government_funding_periods_pkey    | CREATE UNIQUE INDEX government_funding_periods_pkey ON public.government_funding_periods USING btree (id)                       |
| idx_gov_funding_periods_funding_id | CREATE INDEX idx_gov_funding_periods_funding_id ON public.government_funding_periods USING btree (government_funding_id)        |
| idx_gov_funding_periods_period     | CREATE INDEX idx_gov_funding_periods_period ON public.government_funding_periods USING btree (government_funding_id, from_date) |

## Relations

```mermaid
erDiagram

"public.government_funding_properties" }o--|| "public.government_funding_periods" : "FOREIGN KEY (period_id) REFERENCES government_funding_periods(id) ON DELETE CASCADE"
"public.government_funding_periods" }o--|| "public.government_fundings" : "FOREIGN KEY (government_funding_id) REFERENCES government_fundings(id) ON DELETE CASCADE"

"public.government_funding_periods" {
  bigint id
  bigint government_funding_id FK
  date from_date
  date to_date
  double_precision full_time_weekly_hours
  varchar_1000_ comment
  timestamp_with_time_zone created_at
  timestamp_with_time_zone updated_at
}
"public.government_funding_properties" {
  bigint id
  bigint period_id FK
  varchar_100_ key
  varchar_255_ value
  varchar_255_ label
  integer payment
  double_precision requirement
  integer min_age
  integer max_age
  varchar_500_ comment
  timestamp_with_time_zone created_at
  boolean apply_to_all_contracts
}
"public.government_fundings" {
  bigint id
  varchar_255_ name
  varchar_50_ state
  timestamp_with_time_zone created_at
  timestamp_with_time_zone updated_at
}
```

---

> Generated by [tbls](https://github.com/k1LoW/tbls)
