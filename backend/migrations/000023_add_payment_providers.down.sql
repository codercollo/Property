ALTER TABLE payments DROP CONSTRAINT IF EXISTS check_payment_status;
ALTER TABLE payments DROP CONSTRAINT IF EXISTS check_payment_provider;

DROP INDEX IF EXISTS idx_payments_checkout_request_id;
DROP INDEX IF EXISTS idx_payments_transaction_id;

ALTER TABLE payments DROP COLUMN IF EXISTS result_desc;
ALTER TABLE payments DROP COLUMN IF EXISTS result_code;
ALTER TABLE payments DROP COLUMN IF EXISTS checkout_request_id;
ALTER TABLE payments DROP COLUMN IF EXISTS merchant_request_id;
ALTER TABLE payments DROP COLUMN IF EXISTS transaction_desc;
ALTER TABLE payments DROP COLUMN IF EXISTS account_reference;
ALTER TABLE payments DROP COLUMN IF EXISTS phone_number;
ALTER TABLE payments DROP COLUMN IF EXISTS transaction_id;
ALTER TABLE payments DROP COLUMN IF EXISTS payment_provider;