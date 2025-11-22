DROP IF EXISTS TABLE raw_events;

DROP IF EXISTS TABLE users;

DROP IF EXISTS TABLE aggregated_results;

DROP IF EXISTS TABLE ml_results;

DROP IF EXISTS TABLE stat_results;

DROP IF EXISTS INDEX idx_ml_results_session
DROP IF EXISTS INDEX idx_aggregated_results_user
DROP IF EXISTS INDEX idx_stat_results_session