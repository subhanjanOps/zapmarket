DROP TABLE IF EXISTS cod_reconciliations;
DROP TABLE IF EXISTS proof_of_delivery;
DROP TABLE IF EXISTS outbox;

ALTER TABLE shipments
    DROP COLUMN IF EXISTS assigned_agent_id,
    DROP COLUMN IF EXISTS attempt_count,
    DROP COLUMN IF EXISTS next_attempt_at,
    DROP COLUMN IF EXISTS last_attempt_at;

DROP TABLE IF EXISTS delivery_agents;
