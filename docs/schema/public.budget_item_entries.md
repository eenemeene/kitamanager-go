# public.budget_item_entries

## Columns

| Name           | Type                     | Default                                         | Nullable | Children | Parents                                       | Comment |
| -------------- | ------------------------ | ----------------------------------------------- | -------- | -------- | --------------------------------------------- | ------- |
| id             | bigint                   | nextval('budget_item_entries_id_seq'::regclass) | false    |          |                                               |         |
| budget_item_id | bigint                   |                                                 | false    |          | [public.budget_items](public.budget_items.md) |         |
| from_date      | date                     |                                                 | false    |          |                                               |         |
| to_date        | date                     |                                                 | true     |          |                                               |         |
| amount_cents   | integer                  |                                                 | false    |          |                                               |         |
| notes          | varchar(500)             |                                                 | true     |          |                                               |         |
| created_at     | timestamp with time zone |                                                 | true     |          |                                               |         |
| updated_at     | timestamp with time zone |                                                 | true     |          |                                               |         |

## Constraints

| Name                                        | Type        | Definition                                                                                                                      |
| ------------------------------------------- | ----------- | ------------------------------------------------------------------------------------------------------------------------------- |
| budget_item_entries_amount_cents_not_null   | n           | NOT NULL amount_cents                                                                                                           |
| budget_item_entries_amount_nonneg           | CHECK       | CHECK ((amount_cents >= 0))                                                                                                     |
| budget_item_entries_budget_item_id_not_null | n           | NOT NULL budget_item_id                                                                                                         |
| budget_item_entries_from_date_not_null      | n           | NOT NULL from_date                                                                                                              |
| budget_item_entries_id_not_null             | n           | NOT NULL id                                                                                                                     |
| budget_item_entries_budget_item_id_fkey     | FOREIGN KEY | FOREIGN KEY (budget_item_id) REFERENCES budget_items(id) ON DELETE CASCADE                                                      |
| budget_item_entries_pkey                    | PRIMARY KEY | PRIMARY KEY (id)                                                                                                                |
| budget_item_entries_no_overlap              | x           | EXCLUDE USING gist (budget_item_id WITH =, daterange(from_date, COALESCE((to_date + 1), 'infinity'::date), '[)'::text) WITH &&) |

## Indexes

| Name                                   | Definition                                                                                                                                                                         |
| -------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| budget_item_entries_pkey               | CREATE UNIQUE INDEX budget_item_entries_pkey ON public.budget_item_entries USING btree (id)                                                                                        |
| idx_budget_item_entries_budget_item_id | CREATE INDEX idx_budget_item_entries_budget_item_id ON public.budget_item_entries USING btree (budget_item_id)                                                                     |
| idx_budget_item_entries_period         | CREATE INDEX idx_budget_item_entries_period ON public.budget_item_entries USING btree (budget_item_id, from_date)                                                                  |
| budget_item_entries_no_overlap         | CREATE INDEX budget_item_entries_no_overlap ON public.budget_item_entries USING gist (budget_item_id, daterange(from_date, COALESCE((to_date + 1), 'infinity'::date), '[)'::text)) |

## Relations

```mermaid
erDiagram

"public.budget_item_entries" }o--|| "public.budget_items" : "FOREIGN KEY (budget_item_id) REFERENCES budget_items(id) ON DELETE CASCADE"
"public.budget_items" }o--|| "public.organizations" : "FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE"

"public.budget_item_entries" {
  bigint id
  bigint budget_item_id FK
  date from_date
  date to_date
  integer amount_cents
  varchar_500_ notes
  timestamp_with_time_zone created_at
  timestamp_with_time_zone updated_at
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
```

---

> Generated by [tbls](https://github.com/k1LoW/tbls)
