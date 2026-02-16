-- Add missing foreign key constraints.
-- budget_items.organization_id and employee_contracts.pay_plan_id were
-- created without REFERENCES, so referential integrity was not enforced.

ALTER TABLE budget_items
    ADD CONSTRAINT fk_budget_items_organization
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE;

ALTER TABLE employee_contracts
    ADD CONSTRAINT fk_employee_contracts_pay_plan
    FOREIGN KEY (pay_plan_id) REFERENCES pay_plans(id);
