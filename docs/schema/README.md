# kitamanager

## Tables

| Name                                                                                  | Columns | Comment | Type       |
| ------------------------------------------------------------------------------------- | ------- | ------- | ---------- |
| [public.schema_migrations](public.schema_migrations.md)                               | 2       |         | BASE TABLE |
| [public.organizations](public.organizations.md)                                       | 8       |         | BASE TABLE |
| [public.users](public.users.md)                                                       | 11      |         | BASE TABLE |
| [public.user_organizations](public.user_organizations.md)                             | 5       |         | BASE TABLE |
| [public.sections](public.sections.md)                                                 | 10      |         | BASE TABLE |
| [public.employees](public.employees.md)                                               | 8       |         | BASE TABLE |
| [public.pay_plans](public.pay_plans.md)                                               | 5       |         | BASE TABLE |
| [public.pay_plan_periods](public.pay_plan_periods.md)                                 | 8       |         | BASE TABLE |
| [public.pay_plan_entries](public.pay_plan_entries.md)                                 | 8       |         | BASE TABLE |
| [public.employee_contracts](public.employee_contracts.md)                             | 14      |         | BASE TABLE |
| [public.children](public.children.md)                                                 | 9       |         | BASE TABLE |
| [public.child_contracts](public.child_contracts.md)                                   | 9       |         | BASE TABLE |
| [public.government_fundings](public.government_fundings.md)                           | 5       |         | BASE TABLE |
| [public.government_funding_periods](public.government_funding_periods.md)             | 9       |         | BASE TABLE |
| [public.government_funding_properties](public.government_funding_properties.md)       | 12      |         | BASE TABLE |
| [public.child_attendances](public.child_attendances.md)                               | 11      |         | BASE TABLE |
| [public.budget_items](public.budget_items.md)                                         | 7       |         | BASE TABLE |
| [public.budget_item_entries](public.budget_item_entries.md)                           | 8       |         | BASE TABLE |
| [public.audit_logs](public.audit_logs.md)                                             | 13      |         | BASE TABLE |
| [public.government_funding_bill_periods](public.government_funding_bill_periods.md)   | 16      |         | BASE TABLE |
| [public.government_funding_bill_children](public.government_funding_bill_children.md) | 6       |         | BASE TABLE |
| [public.government_funding_bill_payments](public.government_funding_bill_payments.md) | 8       |         | BASE TABLE |
| [public.child_vouchers](public.child_vouchers.md)                                     | 5       |         | BASE TABLE |
| [public.sessions](public.sessions.md)                                                 | 10      |         | BASE TABLE |
| [public.factors](public.factors.md)                                                   | 10      |         | BASE TABLE |
| [public.factor_totp_secrets](public.factor_totp_secrets.md)                           | 4       |         | BASE TABLE |
| [public.factor_backup_codes](public.factor_backup_codes.md)                           | 5       |         | BASE TABLE |
| [public.factor_webauthn_credentials](public.factor_webauthn_credentials.md)           | 11      |         | BASE TABLE |

## Stored procedures and functions

| Name                                | ReturnType    | Arguments                                                      | Type     |
| ----------------------------------- | ------------- | -------------------------------------------------------------- | -------- |
| public.gbtreekey4_in                | gbtreekey4    | cstring                                                        | FUNCTION |
| public.gbtreekey4_out               | cstring       | gbtreekey4                                                     | FUNCTION |
| public.gbtreekey8_in                | gbtreekey8    | cstring                                                        | FUNCTION |
| public.gbtreekey8_out               | cstring       | gbtreekey8                                                     | FUNCTION |
| public.gbtreekey16_in               | gbtreekey16   | cstring                                                        | FUNCTION |
| public.gbtreekey16_out              | cstring       | gbtreekey16                                                    | FUNCTION |
| public.gbtreekey32_in               | gbtreekey32   | cstring                                                        | FUNCTION |
| public.gbtreekey32_out              | cstring       | gbtreekey32                                                    | FUNCTION |
| public.gbtreekey_var_in             | gbtreekey_var | cstring                                                        | FUNCTION |
| public.gbtreekey_var_out            | cstring       | gbtreekey_var                                                  | FUNCTION |
| public.cash_dist                    | money         | money, money                                                   | FUNCTION |
| public.date_dist                    | int4          | date, date                                                     | FUNCTION |
| public.float4_dist                  | float4        | real, real                                                     | FUNCTION |
| public.float8_dist                  | float8        | double precision, double precision                             | FUNCTION |
| public.int2_dist                    | int2          | smallint, smallint                                             | FUNCTION |
| public.int4_dist                    | int4          | integer, integer                                               | FUNCTION |
| public.int8_dist                    | int8          | bigint, bigint                                                 | FUNCTION |
| public.interval_dist                | interval      | interval, interval                                             | FUNCTION |
| public.oid_dist                     | oid           | oid, oid                                                       | FUNCTION |
| public.time_dist                    | interval      | time without time zone, time without time zone                 | FUNCTION |
| public.ts_dist                      | interval      | timestamp without time zone, timestamp without time zone       | FUNCTION |
| public.tstz_dist                    | interval      | timestamp with time zone, timestamp with time zone             | FUNCTION |
| public.gbt_oid_consistent           | bool          | internal, oid, smallint, oid, internal                         | FUNCTION |
| public.gbt_oid_distance             | float8        | internal, oid, smallint, oid, internal                         | FUNCTION |
| public.gbt_oid_fetch                | internal      | internal                                                       | FUNCTION |
| public.gbt_oid_compress             | internal      | internal                                                       | FUNCTION |
| public.gbt_decompress               | internal      | internal                                                       | FUNCTION |
| public.gbt_var_decompress           | internal      | internal                                                       | FUNCTION |
| public.gbt_var_fetch                | internal      | internal                                                       | FUNCTION |
| public.gbt_oid_penalty              | internal      | internal, internal, internal                                   | FUNCTION |
| public.gbt_oid_picksplit            | internal      | internal, internal                                             | FUNCTION |
| public.gbt_oid_union                | gbtreekey8    | internal, internal                                             | FUNCTION |
| public.gbt_oid_same                 | internal      | gbtreekey8, gbtreekey8, internal                               | FUNCTION |
| public.gbt_int2_consistent          | bool          | internal, smallint, smallint, oid, internal                    | FUNCTION |
| public.gbt_int2_distance            | float8        | internal, smallint, smallint, oid, internal                    | FUNCTION |
| public.gbt_int2_compress            | internal      | internal                                                       | FUNCTION |
| public.gbt_int2_fetch               | internal      | internal                                                       | FUNCTION |
| public.gbt_int2_penalty             | internal      | internal, internal, internal                                   | FUNCTION |
| public.gbt_int2_picksplit           | internal      | internal, internal                                             | FUNCTION |
| public.gbt_int2_union               | gbtreekey4    | internal, internal                                             | FUNCTION |
| public.gbt_int2_same                | internal      | gbtreekey4, gbtreekey4, internal                               | FUNCTION |
| public.gbt_int4_consistent          | bool          | internal, integer, smallint, oid, internal                     | FUNCTION |
| public.gbt_int4_distance            | float8        | internal, integer, smallint, oid, internal                     | FUNCTION |
| public.gbt_int4_compress            | internal      | internal                                                       | FUNCTION |
| public.gbt_int4_fetch               | internal      | internal                                                       | FUNCTION |
| public.gbt_int4_penalty             | internal      | internal, internal, internal                                   | FUNCTION |
| public.gbt_int4_picksplit           | internal      | internal, internal                                             | FUNCTION |
| public.gbt_int4_union               | gbtreekey8    | internal, internal                                             | FUNCTION |
| public.gbt_int4_same                | internal      | gbtreekey8, gbtreekey8, internal                               | FUNCTION |
| public.gbt_int8_consistent          | bool          | internal, bigint, smallint, oid, internal                      | FUNCTION |
| public.gbt_int8_distance            | float8        | internal, bigint, smallint, oid, internal                      | FUNCTION |
| public.gbt_int8_compress            | internal      | internal                                                       | FUNCTION |
| public.gbt_int8_fetch               | internal      | internal                                                       | FUNCTION |
| public.gbt_int8_penalty             | internal      | internal, internal, internal                                   | FUNCTION |
| public.gbt_int8_picksplit           | internal      | internal, internal                                             | FUNCTION |
| public.gbt_int8_union               | gbtreekey16   | internal, internal                                             | FUNCTION |
| public.gbt_int8_same                | internal      | gbtreekey16, gbtreekey16, internal                             | FUNCTION |
| public.gbt_float4_consistent        | bool          | internal, real, smallint, oid, internal                        | FUNCTION |
| public.gbt_float4_distance          | float8        | internal, real, smallint, oid, internal                        | FUNCTION |
| public.gbt_float4_compress          | internal      | internal                                                       | FUNCTION |
| public.gbt_float4_fetch             | internal      | internal                                                       | FUNCTION |
| public.gbt_float4_penalty           | internal      | internal, internal, internal                                   | FUNCTION |
| public.gbt_float4_picksplit         | internal      | internal, internal                                             | FUNCTION |
| public.gbt_float4_union             | gbtreekey8    | internal, internal                                             | FUNCTION |
| public.gbt_float4_same              | internal      | gbtreekey8, gbtreekey8, internal                               | FUNCTION |
| public.gbt_float8_consistent        | bool          | internal, double precision, smallint, oid, internal            | FUNCTION |
| public.gbt_float8_distance          | float8        | internal, double precision, smallint, oid, internal            | FUNCTION |
| public.gbt_float8_compress          | internal      | internal                                                       | FUNCTION |
| public.gbt_float8_fetch             | internal      | internal                                                       | FUNCTION |
| public.gbt_float8_penalty           | internal      | internal, internal, internal                                   | FUNCTION |
| public.gbt_float8_picksplit         | internal      | internal, internal                                             | FUNCTION |
| public.gbt_float8_union             | gbtreekey16   | internal, internal                                             | FUNCTION |
| public.gbt_float8_same              | internal      | gbtreekey16, gbtreekey16, internal                             | FUNCTION |
| public.gbt_ts_consistent            | bool          | internal, timestamp without time zone, smallint, oid, internal | FUNCTION |
| public.gbt_ts_distance              | float8        | internal, timestamp without time zone, smallint, oid, internal | FUNCTION |
| public.gbt_tstz_consistent          | bool          | internal, timestamp with time zone, smallint, oid, internal    | FUNCTION |
| public.gbt_tstz_distance            | float8        | internal, timestamp with time zone, smallint, oid, internal    | FUNCTION |
| public.gbt_ts_compress              | internal      | internal                                                       | FUNCTION |
| public.gbt_tstz_compress            | internal      | internal                                                       | FUNCTION |
| public.gbt_ts_fetch                 | internal      | internal                                                       | FUNCTION |
| public.gbt_ts_penalty               | internal      | internal, internal, internal                                   | FUNCTION |
| public.gbt_ts_picksplit             | internal      | internal, internal                                             | FUNCTION |
| public.gbt_ts_union                 | gbtreekey16   | internal, internal                                             | FUNCTION |
| public.gbt_ts_same                  | internal      | gbtreekey16, gbtreekey16, internal                             | FUNCTION |
| public.gbt_time_consistent          | bool          | internal, time without time zone, smallint, oid, internal      | FUNCTION |
| public.gbt_time_distance            | float8        | internal, time without time zone, smallint, oid, internal      | FUNCTION |
| public.gbt_timetz_consistent        | bool          | internal, time with time zone, smallint, oid, internal         | FUNCTION |
| public.gbt_time_compress            | internal      | internal                                                       | FUNCTION |
| public.gbt_timetz_compress          | internal      | internal                                                       | FUNCTION |
| public.gbt_time_fetch               | internal      | internal                                                       | FUNCTION |
| public.gbt_time_penalty             | internal      | internal, internal, internal                                   | FUNCTION |
| public.gbt_time_picksplit           | internal      | internal, internal                                             | FUNCTION |
| public.gbt_time_union               | gbtreekey16   | internal, internal                                             | FUNCTION |
| public.gbt_time_same                | internal      | gbtreekey16, gbtreekey16, internal                             | FUNCTION |
| public.gbt_date_consistent          | bool          | internal, date, smallint, oid, internal                        | FUNCTION |
| public.gbt_date_distance            | float8        | internal, date, smallint, oid, internal                        | FUNCTION |
| public.gbt_date_compress            | internal      | internal                                                       | FUNCTION |
| public.gbt_date_fetch               | internal      | internal                                                       | FUNCTION |
| public.gbt_date_penalty             | internal      | internal, internal, internal                                   | FUNCTION |
| public.gbt_date_picksplit           | internal      | internal, internal                                             | FUNCTION |
| public.gbt_date_union               | gbtreekey8    | internal, internal                                             | FUNCTION |
| public.gbt_date_same                | internal      | gbtreekey8, gbtreekey8, internal                               | FUNCTION |
| public.gbt_intv_consistent          | bool          | internal, interval, smallint, oid, internal                    | FUNCTION |
| public.gbt_intv_distance            | float8        | internal, interval, smallint, oid, internal                    | FUNCTION |
| public.gbt_intv_compress            | internal      | internal                                                       | FUNCTION |
| public.gbt_intv_decompress          | internal      | internal                                                       | FUNCTION |
| public.gbt_intv_fetch               | internal      | internal                                                       | FUNCTION |
| public.gbt_intv_penalty             | internal      | internal, internal, internal                                   | FUNCTION |
| public.gbt_intv_picksplit           | internal      | internal, internal                                             | FUNCTION |
| public.gbt_intv_union               | gbtreekey32   | internal, internal                                             | FUNCTION |
| public.gbt_intv_same                | internal      | gbtreekey32, gbtreekey32, internal                             | FUNCTION |
| public.gbt_cash_consistent          | bool          | internal, money, smallint, oid, internal                       | FUNCTION |
| public.gbt_cash_distance            | float8        | internal, money, smallint, oid, internal                       | FUNCTION |
| public.gbt_cash_compress            | internal      | internal                                                       | FUNCTION |
| public.gbt_cash_fetch               | internal      | internal                                                       | FUNCTION |
| public.gbt_cash_penalty             | internal      | internal, internal, internal                                   | FUNCTION |
| public.gbt_cash_picksplit           | internal      | internal, internal                                             | FUNCTION |
| public.gbt_cash_union               | gbtreekey16   | internal, internal                                             | FUNCTION |
| public.gbt_cash_same                | internal      | gbtreekey16, gbtreekey16, internal                             | FUNCTION |
| public.gbt_macad_consistent         | bool          | internal, macaddr, smallint, oid, internal                     | FUNCTION |
| public.gbt_macad_compress           | internal      | internal                                                       | FUNCTION |
| public.gbt_macad_fetch              | internal      | internal                                                       | FUNCTION |
| public.gbt_macad_penalty            | internal      | internal, internal, internal                                   | FUNCTION |
| public.gbt_macad_picksplit          | internal      | internal, internal                                             | FUNCTION |
| public.gbt_macad_union              | gbtreekey16   | internal, internal                                             | FUNCTION |
| public.gbt_macad_same               | internal      | gbtreekey16, gbtreekey16, internal                             | FUNCTION |
| public.gbt_text_consistent          | bool          | internal, text, smallint, oid, internal                        | FUNCTION |
| public.gbt_bpchar_consistent        | bool          | internal, character, smallint, oid, internal                   | FUNCTION |
| public.gbt_text_compress            | internal      | internal                                                       | FUNCTION |
| public.gbt_bpchar_compress          | internal      | internal                                                       | FUNCTION |
| public.gbt_text_penalty             | internal      | internal, internal, internal                                   | FUNCTION |
| public.gbt_text_picksplit           | internal      | internal, internal                                             | FUNCTION |
| public.gbt_text_union               | gbtreekey_var | internal, internal                                             | FUNCTION |
| public.gbt_text_same                | internal      | gbtreekey_var, gbtreekey_var, internal                         | FUNCTION |
| public.gbt_bytea_consistent         | bool          | internal, bytea, smallint, oid, internal                       | FUNCTION |
| public.gbt_bytea_compress           | internal      | internal                                                       | FUNCTION |
| public.gbt_bytea_penalty            | internal      | internal, internal, internal                                   | FUNCTION |
| public.gbt_bytea_picksplit          | internal      | internal, internal                                             | FUNCTION |
| public.gbt_bytea_union              | gbtreekey_var | internal, internal                                             | FUNCTION |
| public.gbt_bytea_same               | internal      | gbtreekey_var, gbtreekey_var, internal                         | FUNCTION |
| public.gbt_numeric_consistent       | bool          | internal, numeric, smallint, oid, internal                     | FUNCTION |
| public.gbt_numeric_compress         | internal      | internal                                                       | FUNCTION |
| public.gbt_numeric_penalty          | internal      | internal, internal, internal                                   | FUNCTION |
| public.gbt_numeric_picksplit        | internal      | internal, internal                                             | FUNCTION |
| public.gbt_numeric_union            | gbtreekey_var | internal, internal                                             | FUNCTION |
| public.gbt_numeric_same             | internal      | gbtreekey_var, gbtreekey_var, internal                         | FUNCTION |
| public.gbt_bit_consistent           | bool          | internal, bit, smallint, oid, internal                         | FUNCTION |
| public.gbt_bit_compress             | internal      | internal                                                       | FUNCTION |
| public.gbt_bit_penalty              | internal      | internal, internal, internal                                   | FUNCTION |
| public.gbt_bit_picksplit            | internal      | internal, internal                                             | FUNCTION |
| public.gbt_bit_union                | gbtreekey_var | internal, internal                                             | FUNCTION |
| public.gbt_bit_same                 | internal      | gbtreekey_var, gbtreekey_var, internal                         | FUNCTION |
| public.gbt_inet_consistent          | bool          | internal, inet, smallint, oid, internal                        | FUNCTION |
| public.gbt_inet_compress            | internal      | internal                                                       | FUNCTION |
| public.gbt_inet_penalty             | internal      | internal, internal, internal                                   | FUNCTION |
| public.gbt_inet_picksplit           | internal      | internal, internal                                             | FUNCTION |
| public.gbt_inet_union               | gbtreekey16   | internal, internal                                             | FUNCTION |
| public.gbt_inet_same                | internal      | gbtreekey16, gbtreekey16, internal                             | FUNCTION |
| public.gbt_uuid_consistent          | bool          | internal, uuid, smallint, oid, internal                        | FUNCTION |
| public.gbt_uuid_fetch               | internal      | internal                                                       | FUNCTION |
| public.gbt_uuid_compress            | internal      | internal                                                       | FUNCTION |
| public.gbt_uuid_penalty             | internal      | internal, internal, internal                                   | FUNCTION |
| public.gbt_uuid_picksplit           | internal      | internal, internal                                             | FUNCTION |
| public.gbt_uuid_union               | gbtreekey32   | internal, internal                                             | FUNCTION |
| public.gbt_uuid_same                | internal      | gbtreekey32, gbtreekey32, internal                             | FUNCTION |
| public.gbt_macad8_consistent        | bool          | internal, macaddr8, smallint, oid, internal                    | FUNCTION |
| public.gbt_macad8_compress          | internal      | internal                                                       | FUNCTION |
| public.gbt_macad8_fetch             | internal      | internal                                                       | FUNCTION |
| public.gbt_macad8_penalty           | internal      | internal, internal, internal                                   | FUNCTION |
| public.gbt_macad8_picksplit         | internal      | internal, internal                                             | FUNCTION |
| public.gbt_macad8_union             | gbtreekey16   | internal, internal                                             | FUNCTION |
| public.gbt_macad8_same              | internal      | gbtreekey16, gbtreekey16, internal                             | FUNCTION |
| public.gbt_enum_consistent          | bool          | internal, anyenum, smallint, oid, internal                     | FUNCTION |
| public.gbt_enum_compress            | internal      | internal                                                       | FUNCTION |
| public.gbt_enum_fetch               | internal      | internal                                                       | FUNCTION |
| public.gbt_enum_penalty             | internal      | internal, internal, internal                                   | FUNCTION |
| public.gbt_enum_picksplit           | internal      | internal, internal                                             | FUNCTION |
| public.gbt_enum_union               | gbtreekey8    | internal, internal                                             | FUNCTION |
| public.gbt_enum_same                | internal      | gbtreekey8, gbtreekey8, internal                               | FUNCTION |
| public.gbtreekey2_in                | gbtreekey2    | cstring                                                        | FUNCTION |
| public.gbtreekey2_out               | cstring       | gbtreekey2                                                     | FUNCTION |
| public.gbt_bool_consistent          | bool          | internal, boolean, smallint, oid, internal                     | FUNCTION |
| public.gbt_bool_compress            | internal      | internal                                                       | FUNCTION |
| public.gbt_bool_fetch               | internal      | internal                                                       | FUNCTION |
| public.gbt_bool_penalty             | internal      | internal, internal, internal                                   | FUNCTION |
| public.gbt_bool_picksplit           | internal      | internal, internal                                             | FUNCTION |
| public.gbt_bool_union               | gbtreekey2    | internal, internal                                             | FUNCTION |
| public.gbt_bool_same                | internal      | gbtreekey2, gbtreekey2, internal                               | FUNCTION |
| public.gbt_bit_sortsupport          | void          | internal                                                       | FUNCTION |
| public.gbt_varbit_sortsupport       | void          | internal                                                       | FUNCTION |
| public.gbt_bool_sortsupport         | void          | internal                                                       | FUNCTION |
| public.gbt_bytea_sortsupport        | void          | internal                                                       | FUNCTION |
| public.gbt_cash_sortsupport         | void          | internal                                                       | FUNCTION |
| public.gbt_date_sortsupport         | void          | internal                                                       | FUNCTION |
| public.gbt_enum_sortsupport         | void          | internal                                                       | FUNCTION |
| public.gbt_float4_sortsupport       | void          | internal                                                       | FUNCTION |
| public.gbt_float8_sortsupport       | void          | internal                                                       | FUNCTION |
| public.gbt_inet_sortsupport         | void          | internal                                                       | FUNCTION |
| public.gbt_int2_sortsupport         | void          | internal                                                       | FUNCTION |
| public.gbt_int4_sortsupport         | void          | internal                                                       | FUNCTION |
| public.gbt_int8_sortsupport         | void          | internal                                                       | FUNCTION |
| public.gbt_intv_sortsupport         | void          | internal                                                       | FUNCTION |
| public.gbt_macaddr_sortsupport      | void          | internal                                                       | FUNCTION |
| public.gbt_macad8_sortsupport       | void          | internal                                                       | FUNCTION |
| public.gbt_numeric_sortsupport      | void          | internal                                                       | FUNCTION |
| public.gbt_oid_sortsupport          | void          | internal                                                       | FUNCTION |
| public.gbt_text_sortsupport         | void          | internal                                                       | FUNCTION |
| public.gbt_bpchar_sortsupport       | void          | internal                                                       | FUNCTION |
| public.gbt_time_sortsupport         | void          | internal                                                       | FUNCTION |
| public.gbt_ts_sortsupport           | void          | internal                                                       | FUNCTION |
| public.gbt_uuid_sortsupport         | void          | internal                                                       | FUNCTION |
| public.gist_translate_cmptype_btree | int2          | integer                                                        | FUNCTION |

## Relations

```mermaid
erDiagram

"public.user_organizations" }o--|| "public.organizations" : "FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE"
"public.user_organizations" }o--|| "public.users" : "FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE"
"public.sections" }o--|| "public.organizations" : "FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE"
"public.employees" }o--|| "public.organizations" : "FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE"
"public.pay_plans" }o--|| "public.organizations" : "FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE"
"public.pay_plan_periods" }o--|| "public.pay_plans" : "FOREIGN KEY (pay_plan_id) REFERENCES pay_plans(id) ON DELETE CASCADE"
"public.pay_plan_entries" }o--|| "public.pay_plan_periods" : "FOREIGN KEY (period_id) REFERENCES pay_plan_periods(id) ON DELETE CASCADE"
"public.employee_contracts" }o--|| "public.sections" : "FOREIGN KEY (section_id) REFERENCES sections(id)"
"public.employee_contracts" }o--|| "public.employees" : "FOREIGN KEY (employee_id) REFERENCES employees(id) ON DELETE CASCADE"
"public.employee_contracts" }o--|| "public.pay_plans" : "FOREIGN KEY (pay_plan_id) REFERENCES pay_plans(id)"
"public.children" }o--|| "public.organizations" : "FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE"
"public.child_contracts" }o--|| "public.sections" : "FOREIGN KEY (section_id) REFERENCES sections(id)"
"public.child_contracts" }o--|| "public.children" : "FOREIGN KEY (child_id) REFERENCES children(id) ON DELETE CASCADE"
"public.government_funding_periods" }o--|| "public.government_fundings" : "FOREIGN KEY (government_funding_id) REFERENCES government_fundings(id) ON DELETE CASCADE"
"public.government_funding_properties" }o--|| "public.government_funding_periods" : "FOREIGN KEY (period_id) REFERENCES government_funding_periods(id) ON DELETE CASCADE"
"public.child_attendances" }o--|| "public.organizations" : "FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE"
"public.child_attendances" }o--o| "public.users" : "FOREIGN KEY (recorded_by) REFERENCES users(id) ON DELETE SET NULL"
"public.child_attendances" }o--|| "public.children" : "FOREIGN KEY (child_id) REFERENCES children(id) ON DELETE CASCADE"
"public.budget_items" }o--|| "public.organizations" : "FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE"
"public.budget_item_entries" }o--|| "public.budget_items" : "FOREIGN KEY (budget_item_id) REFERENCES budget_items(id) ON DELETE CASCADE"
"public.audit_logs" }o--o| "public.organizations" : "FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE SET NULL"
"public.government_funding_bill_periods" }o--|| "public.organizations" : "FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE"
"public.government_funding_bill_periods" }o--o| "public.users" : "FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL"
"public.government_funding_bill_children" }o--|| "public.government_funding_bill_periods" : "FOREIGN KEY (period_id) REFERENCES government_funding_bill_periods(id) ON DELETE CASCADE"
"public.government_funding_bill_payments" }o--|| "public.government_funding_bill_children" : "FOREIGN KEY (child_id) REFERENCES government_funding_bill_children(id) ON DELETE CASCADE"
"public.child_vouchers" }o--|| "public.children" : "FOREIGN KEY (child_id) REFERENCES children(id) ON DELETE CASCADE"
"public.sessions" }o--|| "public.users" : "FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE"
"public.factors" }o--|| "public.users" : "FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE"
"public.factor_totp_secrets" |o--|| "public.factors" : "FOREIGN KEY (factor_id) REFERENCES factors(id) ON DELETE CASCADE"
"public.factor_backup_codes" }o--|| "public.factors" : "FOREIGN KEY (factor_id) REFERENCES factors(id) ON DELETE CASCADE"
"public.factor_webauthn_credentials" |o--|| "public.factors" : "FOREIGN KEY (factor_id) REFERENCES factors(id) ON DELETE CASCADE"

"public.schema_migrations" {
  bigint version
  boolean dirty
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
"public.pay_plan_entries" {
  bigint id
  bigint period_id FK
  text grade
  integer step
  integer monthly_amount
  integer step_min_years
  timestamp_with_time_zone created_at
  timestamp_with_time_zone updated_at
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
"public.government_fundings" {
  bigint id
  varchar_255_ name
  varchar_50_ state
  timestamp_with_time_zone created_at
  timestamp_with_time_zone updated_at
}
"public.government_funding_periods" {
  bigint id
  bigint government_funding_id FK
  date from_date
  date to_date
  double_precision full_time_weekly_hours
  varchar_1000_ comment
  timestamp_with_time_zone created_at
  timestamp_with_time_zone updated_at
  jsonb required_keys
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
"public.budget_items" {
  bigint id
  bigint organization_id FK
  varchar_255_ name
  varchar_50_ category
  boolean per_child
  timestamp_with_time_zone created_at
  timestamp_with_time_zone updated_at
}
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
"public.government_funding_bill_payments" {
  bigint id
  bigint child_id FK
  varchar_100_ key
  varchar_255_ value
  integer amount
  bigint row_index
  varchar_20_ row_type
  date billing_month
}
"public.child_vouchers" {
  bigint id
  bigint child_id FK
  varchar_17_ voucher_number
  date first_seen
  timestamp_with_time_zone created_at
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
