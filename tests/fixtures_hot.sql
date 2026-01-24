-- Insert items from 8 days ago (Hot Links window is 6-12 days)
INSERT INTO ircLink (user, title, url, clicks, content_type, timestamp) VALUES
('history_buff', 'Ancient Link 1', 'http://old.example.com/1', 10, 'text', datetime('now', '-8 days')),
('history_buff', 'Ancient Link 2', 'http://old.example.com/2', 5, 'text', datetime('now', '-8 days')),
('history_buff', 'Ancient Link 3', 'http://old.example.com/3', 20, 'text', datetime('now', '-9 days'));
