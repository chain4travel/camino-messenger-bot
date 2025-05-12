CREATE TABLE cheque_records (
    cheque_record_id   VARBINARY(32)  NOT NULL PRIMARY KEY,
    from_cm_account    VARBINARY(20)  NOT NULL,
    to_cm_account      VARBINARY(20)  NOT NULL,
    to_bot             VARBINARY(20)  NOT NULL,
    counter            VARBINARY(16)  NOT NULL,
    amount             VARBINARY(16)  NOT NULL,
    created_at         VARBINARY(16)  NOT NULL,
    expires_at         VARBINARY(16)  NOT NULL,
    signature          VARBINARY(64)  NOT NULL,
    tx_id              VARBINARY(32),
    status             TINYINT
);

CREATE TABLE issued_cheque_records (
    cheque_record_id   VARBINARY(32)  NOT NULL PRIMARY KEY,
    counter            VARBINARY(16)  NOT NULL,
    amount             VARBINARY(16)  NOT NULL
);
