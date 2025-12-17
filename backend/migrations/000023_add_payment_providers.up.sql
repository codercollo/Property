ALTER TABLE payments 
ADD COLUMN IF NOT EXISTS payment_provider VARCHAR(50) NOT NULL DEFAULT 'bank',
ADD COLUMN IF NOT EXISTS transaction_id VARCHAR(255),
ADD COLUMN IF NOT EXISTS phone_number VARCHAR(20),
ADD COLUMN IF NOT EXISTS account_reference VARCHAR(100),
ADD COLUMN IF NOT EXISTS transaction_desc TEXT,
ADD COLUMN IF NOT EXISTS merchant_request_id VARCHAR(255),
ADD COLUMN IF NOT EXISTS checkout_request_id VARCHAR(255),
ADD COLUMN IF NOT EXISTS result_code VARCHAR(10),
ADD COLUMN IF NOT EXISTS result_desc TEXT;

CREATE INDEX IF NOT EXISTS idx_payments_transaction_id ON payments(transaction_id);
CREATE INDEX IF NOT EXISTS idx_payments_checkout_request_id ON payments(checkout_request_id);

ALTER TABLE payments DROP CONSTRAINT IF EXISTS check_payment_provider;
ALTER TABLE payments ADD CONSTRAINT check_payment_provider CHECK (payment_provider IN ('mpesa', 'bank', 'card'));

ALTER TABLE payments DROP CONSTRAINT IF EXISTS check_payment_status;
ALTER TABLE payments ADD CONSTRAINT check_payment_status CHECK (status IN ('pending', 'completed', 'failed', 'cancelled'));