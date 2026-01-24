-- Insert items from 8 days ago (Hot Links window is 6-12 days)
INSERT INTO ircLink (user, title, url, clicks, content_type, timestamp) VALUES
('history_buff', 'Ancient Link 1', 'http://old.example.com/1', 10, 'text', DATE_SUB(NOW(), INTERVAL 8 DAY)),
('history_buff', 'Ancient Link 2', 'http://old.example.com/2', 5, 'text', DATE_SUB(NOW(), INTERVAL 8 DAY)),
('history_buff', 'Ancient Link 3', 'http://old.example.com/3', 20, 'text', DATE_SUB(NOW(), INTERVAL 9 DAY));
