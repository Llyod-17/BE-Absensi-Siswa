CREATE TABLE notification_logs (
    id BIGINT NOT NULL PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    phone VARCHAR(20),
    status VARCHAR(20),
    message TEXT,
    sent_date DATE NOT NULL,
    sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    response_status VARCHAR(255),

    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE KEY unique_daily_notif (user_id, sent_date)
);
