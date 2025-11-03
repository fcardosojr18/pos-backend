CREATE TABLE IF NOT EXISTS orders (
  id SERIAL PRIMARY KEY,
  subtotal_cents INTEGER NOT NULL,
  tax_cents INTEGER NOT NULL,
  tip_cents INTEGER NOT NULL DEFAULT 0,
  total_cents INTEGER NOT NULL,
  status TEXT NOT NULL DEFAULT 'open',
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at);
