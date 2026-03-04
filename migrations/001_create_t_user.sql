CREATE TABLE t_user (
    v_id BIGSERIAL PRIMARY KEY,
    v_name TEXT NOT NULL,
    v_balance NUMERIC(20,8) NOT NULL DEFAULT 0,
    v_created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO t_user (v_name, v_balance) VALUES ('default', 1000.0);