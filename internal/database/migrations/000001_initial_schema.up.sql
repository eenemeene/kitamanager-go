-- Consolidated initial schema (squashed from migrations 000001-000010).

CREATE TABLE IF NOT EXISTS organizations (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT true,
    state VARCHAR(50) NOT NULL DEFAULT 'berlin',
    created_at TIMESTAMPTZ,
    created_by VARCHAR(255),
    updated_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_organizations_name ON organizations(name);

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT true,
    is_superadmin BOOLEAN DEFAULT false,
    last_login TIMESTAMPTZ,
    created_at TIMESTAMPTZ,
    created_by VARCHAR(255),
    updated_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email);

CREATE TABLE IF NOT EXISTS user_organizations (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255),
    PRIMARY KEY (user_id, organization_id)
);

CREATE TABLE IF NOT EXISTS sections (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    is_default BOOLEAN DEFAULT false,
    min_age_months INTEGER,
    max_age_months INTEGER,
    created_at TIMESTAMPTZ,
    created_by VARCHAR(255),
    updated_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_sections_organization_id ON sections(organization_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_section_org_name ON sections(name, organization_id);

CREATE TABLE IF NOT EXISTS employees (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255) NOT NULL,
    gender VARCHAR(20) NOT NULL,
    birthdate DATE NOT NULL,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_employees_organization_id ON employees(organization_id);

CREATE TABLE IF NOT EXISTS pay_plans (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id),
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_pay_plans_organization_id ON pay_plans(organization_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_pay_plans_org_name ON pay_plans(organization_id, name);

CREATE TABLE IF NOT EXISTS pay_plan_periods (
    id BIGSERIAL PRIMARY KEY,
    pay_plan_id BIGINT NOT NULL REFERENCES pay_plans(id) ON DELETE CASCADE,
    from_date DATE NOT NULL,
    to_date DATE,
    weekly_hours DOUBLE PRECISION NOT NULL,
    employer_contribution_rate INTEGER,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_pay_plan_periods_pay_plan_id ON pay_plan_periods(pay_plan_id);
CREATE INDEX IF NOT EXISTS idx_pay_plan_periods_period ON pay_plan_periods(pay_plan_id, from_date);

CREATE TABLE IF NOT EXISTS pay_plan_entries (
    id BIGSERIAL PRIMARY KEY,
    period_id BIGINT NOT NULL REFERENCES pay_plan_periods(id) ON DELETE CASCADE,
    grade TEXT NOT NULL,
    step INTEGER NOT NULL,
    monthly_amount INTEGER NOT NULL,
    step_min_years INTEGER,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_pay_plan_entries_period_id ON pay_plan_entries(period_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_pay_plan_entries_lookup ON pay_plan_entries(period_id, grade, step);

CREATE TABLE IF NOT EXISTS employee_contracts (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    from_date DATE NOT NULL,
    to_date DATE,
    section_id BIGINT NOT NULL REFERENCES sections(id),
    properties JSONB,
    staff_category VARCHAR(50) NOT NULL DEFAULT 'qualified',
    grade VARCHAR(20),
    step INTEGER,
    weekly_hours DOUBLE PRECISION,
    pay_plan_id BIGINT NOT NULL REFERENCES pay_plans(id),
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_employee_contracts_employee_id ON employee_contracts(employee_id);
CREATE INDEX IF NOT EXISTS idx_employee_contracts_section_id ON employee_contracts(section_id);
CREATE INDEX IF NOT EXISTS idx_employee_contracts_pay_plan_id ON employee_contracts(pay_plan_id);
CREATE INDEX IF NOT EXISTS idx_employee_contracts_period ON employee_contracts(employee_id, from_date, to_date);

CREATE TABLE IF NOT EXISTS children (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255) NOT NULL,
    gender VARCHAR(20) NOT NULL,
    birthdate DATE NOT NULL,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_children_organization_id ON children(organization_id);

CREATE TABLE IF NOT EXISTS child_contracts (
    id BIGSERIAL PRIMARY KEY,
    child_id BIGINT NOT NULL REFERENCES children(id) ON DELETE CASCADE,
    from_date DATE NOT NULL,
    to_date DATE,
    section_id BIGINT NOT NULL REFERENCES sections(id),
    properties JSONB,
    voucher_number VARCHAR(50),
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_child_contracts_child_id ON child_contracts(child_id);
CREATE INDEX IF NOT EXISTS idx_child_contracts_section_id ON child_contracts(section_id);
CREATE INDEX IF NOT EXISTS idx_child_contracts_period ON child_contracts(child_id, from_date, to_date);
CREATE INDEX IF NOT EXISTS idx_child_contracts_voucher ON child_contracts(voucher_number);

CREATE TABLE IF NOT EXISTS government_fundings (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    state VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_government_fundings_state ON government_fundings(state);

CREATE TABLE IF NOT EXISTS government_funding_periods (
    id BIGSERIAL PRIMARY KEY,
    government_funding_id BIGINT NOT NULL REFERENCES government_fundings(id) ON DELETE CASCADE,
    from_date DATE NOT NULL,
    to_date DATE,
    full_time_weekly_hours DOUBLE PRECISION NOT NULL,
    comment VARCHAR(1000),
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_gov_funding_periods_funding_id ON government_funding_periods(government_funding_id);
CREATE INDEX IF NOT EXISTS idx_gov_funding_periods_period ON government_funding_periods(government_funding_id, from_date);

CREATE TABLE IF NOT EXISTS government_funding_properties (
    id BIGSERIAL PRIMARY KEY,
    period_id BIGINT NOT NULL REFERENCES government_funding_periods(id) ON DELETE CASCADE,
    key VARCHAR(100) NOT NULL,
    value VARCHAR(255) NOT NULL,
    label VARCHAR(255) NOT NULL,
    payment INTEGER NOT NULL,
    requirement DOUBLE PRECISION NOT NULL,
    min_age INTEGER,
    max_age INTEGER,
    comment VARCHAR(500),
    created_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_gov_funding_properties_period_id ON government_funding_properties(period_id);

CREATE TABLE IF NOT EXISTS child_attendances (
    id BIGSERIAL PRIMARY KEY,
    child_id BIGINT NOT NULL REFERENCES children(id) ON DELETE CASCADE,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    check_in_time TIMESTAMPTZ,
    check_out_time TIMESTAMPTZ,
    status VARCHAR(20) NOT NULL DEFAULT 'present',
    note VARCHAR(500),
    recorded_by BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_child_attendances_child_id ON child_attendances(child_id);
CREATE INDEX IF NOT EXISTS idx_child_attendances_organization_id ON child_attendances(organization_id);
CREATE INDEX IF NOT EXISTS idx_child_attendances_date ON child_attendances(date);
CREATE UNIQUE INDEX IF NOT EXISTS idx_child_attendances_child_date ON child_attendances(child_id, date);
CREATE INDEX IF NOT EXISTS idx_child_attendances_org_date ON child_attendances(organization_id, date);

CREATE TABLE IF NOT EXISTS budget_items (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(50) NOT NULL,
    per_child BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_budget_item_org_name ON budget_items(organization_id, name);

CREATE TABLE IF NOT EXISTS budget_item_entries (
    id BIGSERIAL PRIMARY KEY,
    budget_item_id BIGINT NOT NULL REFERENCES budget_items(id) ON DELETE CASCADE,
    from_date DATE NOT NULL,
    to_date DATE,
    amount_cents INTEGER NOT NULL,
    notes VARCHAR(500),
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_budget_item_entries_budget_item_id ON budget_item_entries(budget_item_id);
CREATE INDEX IF NOT EXISTS idx_budget_item_entries_period ON budget_item_entries(budget_item_id, from_date);

CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    user_id BIGINT,
    user_email VARCHAR(255),
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100),
    resource_id BIGINT,
    ip_address VARCHAR(45),
    user_agent VARCHAR(512),
    details TEXT,
    success BOOLEAN NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);

CREATE TABLE IF NOT EXISTS revoked_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_revoked_tokens_token_hash ON revoked_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_revoked_tokens_user_id ON revoked_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_revoked_tokens_expires_at ON revoked_tokens(expires_at);

CREATE TABLE IF NOT EXISTS government_funding_bill_periods (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id),
    from_date DATE NOT NULL,
    to_date DATE,
    file_name VARCHAR(255) NOT NULL,
    file_sha256 VARCHAR(64) NOT NULL,
    facility_name VARCHAR(255) NOT NULL,
    facility_total INT NOT NULL,
    contract_booking INT NOT NULL,
    correction_booking INT NOT NULL,
    created_by BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_gfbp_org ON government_funding_bill_periods(organization_id);

CREATE TABLE IF NOT EXISTS government_funding_bill_children (
    id BIGSERIAL PRIMARY KEY,
    period_id BIGINT NOT NULL REFERENCES government_funding_bill_periods(id) ON DELETE CASCADE,
    voucher_number VARCHAR(20) NOT NULL,
    child_name VARCHAR(255) NOT NULL,
    birth_date VARCHAR(10) NOT NULL,
    district INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_gfbc_period ON government_funding_bill_children(period_id);

CREATE TABLE IF NOT EXISTS government_funding_bill_payments (
    id BIGSERIAL PRIMARY KEY,
    child_id BIGINT NOT NULL REFERENCES government_funding_bill_children(id) ON DELETE CASCADE,
    key VARCHAR(100) NOT NULL,
    value VARCHAR(255) NOT NULL,
    amount INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_gfbpay_child ON government_funding_bill_payments(child_id);
