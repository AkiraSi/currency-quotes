-- +goose Up
CREATE TABLE actual_rates (
                              base_currency     VARCHAR(3)    NOT NULL,
                              quote_currency    VARCHAR(3)    NOT NULL,
                              price             NUMERIC(20,8) NOT NULL,
                              source_updated_at TIMESTAMPTZ   NOT NULL,
                              updated_at        TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,

                              CONSTRAINT actual_rates_pk PRIMARY KEY (base_currency, quote_currency),
                              CONSTRAINT actual_rates_price_positive CHECK (price > 0)
);

CREATE TABLE quote_updates (
                               update_id                   UUID        NOT NULL,
                               status                      VARCHAR(16) NOT NULL DEFAULT 'accepted',
                               result_price                NUMERIC(20,8),
                               source_updated_at           TIMESTAMPTZ,
                               error_message               VARCHAR(512),
                               created_at                  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
                               started_at                  TIMESTAMPTZ,
                               completed_at                TIMESTAMPTZ,
                               actual_rates_base_currency  VARCHAR(3)  NOT NULL,
                               actual_rates_quote_currency VARCHAR(3)  NOT NULL,

                               CONSTRAINT quote_updates_pk PRIMARY KEY (update_id),
                               CONSTRAINT quote_updates_status_check
                                   CHECK (status IN ('accepted', 'running', 'succeeded', 'failed')),
                               CONSTRAINT quote_updates_result_price_positive
                                   CHECK (result_price IS NULL OR result_price > 0),
                               CONSTRAINT quote_updates_actual_rates_fk
                                   FOREIGN KEY (actual_rates_base_currency, actual_rates_quote_currency)
                                       REFERENCES actual_rates (base_currency, quote_currency)
);

CREATE INDEX quote_updates_status_created_at_idx
    ON quote_updates (status, created_at);

-- +goose Down
DROP TABLE quote_updates;
DROP TABLE actual_rates;
