CREATE TABLE bots (
    cm_account      VARBINARY(20)  NOT NULL,
    bot             VARBINARY(20)  NOT NULL,
    status          INTEGER        NOT NULL,
    PRIMARY KEY (cm_account, bot)
);
