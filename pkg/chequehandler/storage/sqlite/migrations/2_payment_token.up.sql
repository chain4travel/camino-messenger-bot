ALTER TABLE cheque_records
ADD payment_token VARBINARY(20) NOT NULL DEFAULT X'0000000000000000000000000000000000000000';

<-- TODO@ also fix bot logic to ignore 0x00 cheques, since for now we can only cash in erc-20
// TODO@ add removed attributes in activities back