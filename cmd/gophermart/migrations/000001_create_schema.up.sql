CREATE TABLE IF NOT EXISTS users (
    id serial PRIMARY KEY, 
    login varchar(50) NOT NULL, 
    password varchar(50) NOT NULL
    );

CREATE TABLE IF NOT EXISTS orders (
    id serial PRIMARY KEY, 
    user_id bigint NOT NULL, 
    number varchar(100) NOT NULL, 
    status varchar(50) NOT NULL, 
    accrual numeric, 
    uploaded_at timestamptz NOT NULL
    );

CREATE TABLE IF NOT EXISTS balance (
    id serial PRIMARY KEY, 
    user_id bigint NOT NULL, 
    current numeric NOT NULL, 
    withdrawn numeric NOT NULL
    );

CREATE TABLE IF NOT EXISTS withdrawals (
    id serial PRIMARY KEY, 
    user_id bigint NOT NULL, 
    order_id bigint NOT NULL, 
    sum numeric NOT NULL, 
    processed_at timestamptz NOT NULL
    );

ALTER TABLE "orders" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "balance" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "withdrawals" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "withdrawals" ADD FOREIGN KEY ("order_id") REFERENCES "orders" ("id");