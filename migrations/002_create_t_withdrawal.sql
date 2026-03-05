CREATE TABLE t_withdrawal (
    v_id BIGSERIAL PRIMARY KEY,
    v_user_id BIGINT NOT NULL REFERENCES t_user(v_id),
    v_amount NUMERIC(20,8) NOT NULL,
    v_currency TEXT NOT NULL DEFAULT 'USDT',
    v_destination TEXT NOT NULL,
    v_status TEXT NOT NULL DEFAULT 'pending',
    v_idempotency_key TEXT NOT NULL,
    v_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_idempotency_key UNIQUE (v_idempotency_key)
);

CREATE TABLE t_ledger_entry (
    v_id BIGSERIAL PRIMARY KEY,
    v_user_id BIGINT NOT NULL REFERENCES t_user(v_id),
    v_withdrawal_id BIGINT REFERENCES t_withdrawal(v_id),
    v_type TEXT NOT NULL,
    v_amount NUMERIC(20,8) NOT NULL,
    v_balance_after NUMERIC(20,8) NOT NULL,
    v_created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);