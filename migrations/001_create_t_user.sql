CREATE TABLE t_user (
    v_id SERIAL PRIMARY KEY,
    v_name TEXT NOT NULL,
    v_balance NUMERIC(20,8) NOT NULL DEFAULT 0
);

INSERT INTO t_user (v_name, v_balance) VALUES ('default', 1000.0);